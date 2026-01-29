use crate::{PendingFlashBlock, pending_state::PendingBlockState, tx_cache::TransactionCache};
use alloy_eips::{BlockNumberOrTag, eip2718::WithEncoded};
use alloy_primitives::B256;
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use reth_chain_state::{ComputedTrieData, ExecutedBlock};
use reth_errors::RethError;
use reth_evm::{
    ConfigureEvm,
    execute::{BlockBuilder, BlockBuilderOutcome},
};
use reth_execution_types::BlockExecutionOutput;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, Recovered, SealedHeader,
    transaction::TxHashRef,
};
use reth_revm::{cached::CachedReads, database::StateProviderDatabase, db::State};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory, noop::NoopProvider};
use std::{
    sync::Arc,
    time::{Duration, Instant},
};
use tracing::trace;

/// The `FlashBlockBuilder` builds [`PendingBlock`] out of a sequence of transactions.
#[derive(Debug)]
pub(crate) struct FlashBlockBuilder<EvmConfig, Provider> {
    evm_config: EvmConfig,
    provider: Provider,
}

impl<EvmConfig, Provider> FlashBlockBuilder<EvmConfig, Provider> {
    pub(crate) const fn new(evm_config: EvmConfig, provider: Provider) -> Self {
        Self { evm_config, provider }
    }

    pub(crate) const fn provider(&self) -> &Provider {
        &self.provider
    }
}

pub(crate) struct BuildArgs<I, N: NodePrimitives> {
    pub(crate) base: OpFlashblockPayloadBase,
    pub(crate) transactions: I,
    pub(crate) cached_state: Option<(B256, CachedReads)>,
    pub(crate) last_flashblock_index: u64,
    pub(crate) last_flashblock_hash: B256,
    pub(crate) compute_state_root: bool,
    /// Optional pending parent state for speculative building.
    /// When set, allows building on top of a pending block that hasn't been
    /// canonicalized yet.
    pub(crate) pending_parent: Option<PendingBlockState<N>>,
}

/// Result of a flashblock build operation.
#[derive(Debug)]
pub(crate) struct BuildResult<N: NodePrimitives> {
    /// The built pending flashblock.
    pub(crate) pending_flashblock: PendingFlashBlock<N>,
    /// Cached reads from this build.
    pub(crate) cached_reads: CachedReads,
    /// Pending state that can be used for building subsequent blocks.
    pub(crate) pending_state: PendingBlockState<N>,
}

impl<N, EvmConfig, Provider> FlashBlockBuilder<EvmConfig, Provider>
where
    N: NodePrimitives,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<OpFlashblockPayloadBase> + Unpin>,
    Provider: StateProviderFactory
        + BlockReaderIdExt<
            Header = HeaderTy<N>,
            Block = BlockTy<N>,
            Transaction = N::SignedTx,
            Receipt = ReceiptTy<N>,
        > + Unpin,
{
    /// Returns the [`PendingFlashBlock`] made purely out of transactions and
    /// [`OpFlashblockPayloadBase`] in `args`.
    ///
    /// This method supports two building modes:
    /// 1. **Canonical mode**: Parent matches local tip - uses state from storage
    /// 2. **Speculative mode**: Parent is a pending block - uses pending state
    ///
    /// When a `tx_cache` is provided and we're in canonical mode, the builder will
    /// attempt to resume from cached state if the transaction list is a continuation
    /// of what was previously executed.
    ///
    /// Returns `None` if:
    /// - In canonical mode: flashblock doesn't attach to the latest header
    /// - In speculative mode: no pending parent state provided
    pub(crate) fn execute<I: IntoIterator<Item = WithEncoded<Recovered<N::SignedTx>>>>(
        &self,
        mut args: BuildArgs<I, N>,
        tx_cache: Option<&mut TransactionCache<N>>,
    ) -> eyre::Result<Option<BuildResult<N>>> {
        trace!(target: "flashblocks", "Attempting new pending block from flashblocks");

        let latest = self
            .provider
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;
        let latest_hash = latest.hash();

        // Determine build mode: canonical (parent is local tip) or speculative (parent is pending)
        let is_canonical = args.base.parent_hash == latest_hash;
        let has_pending_parent = args.pending_parent.is_some();

        if !is_canonical && !has_pending_parent {
            trace!(
                target: "flashblocks",
                flashblock_parent = ?args.base.parent_hash,
                local_latest = ?latest.num_hash(),
                "Skipping non-consecutive flashblock (no pending parent available)"
            );
            return Ok(None);
        }

        // Collect transactions and extract hashes for cache lookup
        let transactions: Vec<_> = args.transactions.into_iter().collect();
        let tx_hashes: Vec<B256> = transactions.iter().map(|tx| *tx.tx_hash()).collect();

        // Get state provider and parent header context.
        // For speculative builds, use the canonical anchor hash (not the pending parent hash)
        // for storage reads, but execute with the pending parent's sealed header context.
        let (state_provider, canonical_anchor, parent_header) = if is_canonical {
            (self.provider.history_by_block_hash(latest.hash())?, latest.hash(), &latest)
        } else {
            // For speculative building, we need to use the canonical anchor
            // and apply the pending state's bundle on top of it
            let pending = args.pending_parent.as_ref().unwrap();
            let Some(parent_header) = pending.sealed_header.as_ref() else {
                trace!(
                    target: "flashblocks",
                    pending_block_number = pending.block_number,
                    pending_block_hash = ?pending.block_hash,
                    "Skipping speculative build: pending parent header is unavailable"
                );
                return Ok(None);
            };
            trace!(
                target: "flashblocks",
                pending_block_number = pending.block_number,
                pending_block_hash = ?pending.block_hash,
                canonical_anchor = ?pending.canonical_anchor_hash,
                "Building speculatively on pending state"
            );
            (
                self.provider.history_by_block_hash(pending.canonical_anchor_hash)?,
                pending.canonical_anchor_hash,
                parent_header,
            )
        };

        // Set up cached reads
        let cache_key = if is_canonical { latest_hash } else { args.base.parent_hash };
        let mut request_cache = args
            .cached_state
            .take()
            .filter(|(hash, _)| hash == &cache_key)
            .map(|(_, state)| state)
            .unwrap_or_else(|| {
                // For speculative builds, use cached reads from pending parent
                args.pending_parent.as_ref().map(|p| p.cached_reads.clone()).unwrap_or_default()
            });

        let cached_db = request_cache.as_db_mut(StateProviderDatabase::new(&state_provider));

        // NOTE on transaction caching:
        // We intentionally do NOT use cached bundle state as a prestate for canonical builds.
        // The cached bundle represents state AFTER executing transactions, so using it as
        // prestate and re-executing the same transactions would cause failures (nonce already
        // incremented, balance already deducted, etc.).
        //
        // The cache is still updated after builds (for future checkpointing support), but
        // currently only provides logging information about cache hits.
        //
        // Future optimization: reth would need to support "block builder checkpointing" to
        // properly resume partial builds by skipping already-executed transactions AND
        // reconstructing the block body from cached data.
        if is_canonical
            && let Some(cache) = tx_cache.as_ref()
            && let Some((_, _, cached_count)) =
                cache.get_resumable_state(args.base.block_number, &tx_hashes)
        {
            trace!(
                target: "flashblocks",
                cached_tx_count = cached_count,
                total_txs = tx_hashes.len(),
                "Cache hit (prestate not used - all txs will be re-executed)"
            );
        }

        // Build state with appropriate prestate
        // Only speculative builds use bundle prestate (parent block's execution output)
        let mut state = if let Some(ref pending) = args.pending_parent {
            // Speculative mode - pending parent's bundle as prestate
            // This is correct because we're building a NEW block on top of the parent,
            // not re-executing the same transactions
            State::builder()
                .with_database(cached_db)
                .with_bundle_prestate(pending.execution_outcome.state.clone())
                .with_bundle_update()
                .build()
        } else {
            // Canonical mode - fresh build from database state
            // We don't use cached bundle as prestate since we're re-executing the same txs
            State::builder().with_database(cached_db).with_bundle_update().build()
        };

        let mut builder = self
            .evm_config
            .builder_for_next_block(&mut state, parent_header, args.base.clone().into())
            .map_err(RethError::other)?;

        builder.apply_pre_execution_changes()?;

        // Execute all transactions - we cannot skip any because BlockBuilder requires
        // all transactions to be executed to include them in the block body
        for tx in transactions {
            let _gas_used = builder.execute_transaction(tx)?;
        }

        // if the real state root should be computed
        let BlockBuilderOutcome { execution_result, block, hashed_state, .. } =
            if args.compute_state_root {
                trace!(target: "flashblocks", "Computing block state root");
                builder.finish(&state_provider)?
            } else {
                builder.finish(NoopProvider::default())?
            };

        // Take the bundle before creating execution_outcome (for cache update)
        let bundle = state.take_bundle();

        // Update transaction cache if provided (only in canonical mode)
        // Cache the bundle state and receipts for potential reuse as prestate
        if let Some(cache) = tx_cache
            && is_canonical
        {
            cache.update(
                args.base.block_number,
                tx_hashes,
                bundle.clone(),
                execution_result.receipts.clone(),
            );
        }

        let execution_outcome = BlockExecutionOutput { state: bundle, result: execution_result };
        let execution_outcome = Arc::new(execution_outcome);

        // Create pending state for subsequent builds.
        // Use sequencer-provided block hash for parent matching and keep the locally
        // built sealed header for speculative child execution context.
        let sealed_header = SealedHeader::new(block.header().clone(), block.hash());
        let pending_state = PendingBlockState::new(
            args.last_flashblock_hash,
            block.number(),
            args.base.parent_hash,
            canonical_anchor,
            execution_outcome.clone(),
            request_cache.clone(),
        )
        .with_sealed_header(sealed_header);

        let pending_block = PendingBlock::with_executed_block(
            Instant::now() + Duration::from_secs(1),
            ExecutedBlock::new(
                block.into(),
                execution_outcome,
                ComputedTrieData::without_trie_input(
                    Arc::new(hashed_state.into_sorted()),
                    Arc::default(),
                ),
            ),
        );
        let pending_flashblock = PendingFlashBlock::new(
            pending_block,
            args.last_flashblock_index,
            args.last_flashblock_hash,
            args.compute_state_root,
        );

        Ok(Some(BuildResult { pending_flashblock, cached_reads: request_cache, pending_state }))
    }
}

impl<EvmConfig: Clone, Provider: Clone> Clone for FlashBlockBuilder<EvmConfig, Provider> {
    fn clone(&self) -> Self {
        Self { evm_config: self.evm_config.clone(), provider: self.provider.clone() }
    }
}
