// Package Interop
// Dafny module Interop compiled into Go

package Interop

import (
	m_ChainContainer "ChainContainer"
	m_LogsDB "LogsDB"
	m__System "System_"
	m_Types "Types"
	m_Utils "Utils"
	m_VerifiedDB "VerifiedDB"
	_dafny "dafny"
	os "os"
)

var _ = os.Args
var _ _dafny.Dummy__
var _ m__System.Dummy__
var _ m_Types.Dummy__
var _ m_Utils.Dummy__
var _ m_VerifiedDB.Dummy__
var _ m_ChainContainer.Dummy__
var _ m_LogsDB.Dummy__

type Dummy__ struct{}

// Definition of class FrontierView
type FrontierView struct {
	dummy byte
}

func New_FrontierView_() *FrontierView {
	_this := FrontierView{}

	return &_this
}

type CompanionStruct_FrontierView_ struct {
}

var Companion_FrontierView_ = CompanionStruct_FrontierView_{}

func (_this *FrontierView) Equals(other *FrontierView) bool {
	return _this == other
}

func (_this *FrontierView) EqualsGeneric(x interface{}) bool {
	other, ok := x.(*FrontierView)
	return ok && _this.Equals(other)
}

func (*FrontierView) String() string {
	return "Interop.FrontierView"
}

func Type_FrontierView_() _dafny.TypeDescriptor {
	return type_FrontierView_{}
}

type type_FrontierView_ struct {
}

func (_this type_FrontierView_) Default() interface{} {
	return (*FrontierView)(nil)
}

func (_this type_FrontierView_) String() string {
	return "Interop.FrontierView"
}
func (_this *FrontierView) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = &FrontierView{}

func (_this *FrontierView) Contains(chainID _dafny.Int, query m_Types.ContainsQuery) bool {
	{
		return (((((((_this).BlockInfo(chainID)).Dtor_id()).Dtor_number()).Cmp((query).Dtor_blockNum()) == 0) && ((((_this).BlockInfo(chainID)).Dtor_timestamp()).Cmp((query).Dtor_timestamp()) == 0)) && (((query).Dtor_logIdx()).Cmp(_dafny.IntOfUint32((((_this).BlockLogs(chainID)).Dtor_fullLogs()).Cardinality())) < 0)) && ((((((_this).BlockLogs(chainID)).Dtor_fullLogs()).Select(((query).Dtor_logIdx()).Uint32()).(m_Types.Log)).Dtor_checksum()).Cmp((query).Dtor_checksum()) == 0)
	}
}
func (_this *FrontierView) BlockInfo(chainID _dafny.Int) m_Types.BlockInfo {
	{
		return m_Types.Companion_BlockInfo_.Create_BlockInfo_(m_Types.Companion_BlockID_.Create_BlockID_(_dafny.Zero, _dafny.Zero), _dafny.Zero, _dafny.Zero)
	}
}
func (_this *FrontierView) BlockLogs(chainID _dafny.Int) m_Types.BlockLogs {
	{
		return m_Types.Companion_BlockLogs_.Create_BlockLogs_(_dafny.SeqOf(), _dafny.NewMapBuilder().ToMap())
	}
}

// End of class FrontierView

// Definition of class Interop
type Interop struct {
	CurrentL1            m_Types.BlockID
	_verifiedDB          *m_VerifiedDB.VerifiedDB
	_logsDBs             _dafny.Map
	_activationTimestamp _dafny.Int
	_messageExpiryWindow _dafny.Int
	_chains              _dafny.Map
}

func New_Interop_() *Interop {
	_this := Interop{}

	_this.CurrentL1 = m_Types.Companion_BlockID_.Default()
	_this._verifiedDB = (*m_VerifiedDB.VerifiedDB)(nil)
	_this._logsDBs = _dafny.EmptyMap
	_this._activationTimestamp = _dafny.Zero
	_this._messageExpiryWindow = _dafny.Zero
	_this._chains = _dafny.EmptyMap
	return &_this
}

type CompanionStruct_Interop_ struct {
}

var Companion_Interop_ = CompanionStruct_Interop_{}

func (_this *Interop) Equals(other *Interop) bool {
	return _this == other
}

func (_this *Interop) EqualsGeneric(x interface{}) bool {
	other, ok := x.(*Interop)
	return ok && _this.Equals(other)
}

func (*Interop) String() string {
	return "Interop.Interop"
}

func Type_Interop_() _dafny.TypeDescriptor {
	return type_Interop_{}
}

type type_Interop_ struct {
}

func (_this type_Interop_) Default() interface{} {
	return (*Interop)(nil)
}

func (_this type_Interop_) String() string {
	return "Interop.Interop"
}
func (_this *Interop) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = &Interop{}

func (_this *Interop) Ctor__(chains _dafny.Map) {
	{
		var _0_chainIDs _dafny.Sequence
		_ = _0_chainIDs
		var _out0 _dafny.Sequence
		_ = _out0
		_out0 = m_Types.Companion_Default___.Enumerate(m_Types.Companion_Default___.CHAIN__IDS())
		_0_chainIDs = _out0
		var _1_logsDBs _dafny.Map
		_ = _1_logsDBs
		_1_logsDBs = _dafny.NewMapBuilder().ToMap()
		var _hi0 _dafny.Int = _dafny.IntOfUint32((_0_chainIDs).Cardinality())
		_ = _hi0
		for _2_i := _dafny.Zero; _2_i.Cmp(_hi0) < 0; _2_i = _2_i.Plus(_dafny.One) {
			var _3_db *m_LogsDB.LogsDB
			_ = _3_db
			var _nw0 *m_LogsDB.LogsDB = m_LogsDB.New_LogsDB_()
			_ = _nw0
			_nw0.Ctor__()
			_3_db = _nw0
			_1_logsDBs = (_1_logsDBs).Update((_0_chainIDs).Select((_2_i).Uint32()).(_dafny.Int), _3_db)
		}
		var _nw1 *m_VerifiedDB.VerifiedDB = m_VerifiedDB.New_VerifiedDB_()
		_ = _nw1
		_nw1.Ctor__()
		(_this)._verifiedDB = _nw1
		(_this)._activationTimestamp = m_Types.Companion_Default___.ACTIVATION__TIMESTAMP()
		(_this)._messageExpiryWindow = m_Types.Companion_Default___.MESSAGE__EXPIRY__WINDOW()
		(_this).CurrentL1 = m_Types.Companion_BlockID_.Create_BlockID_(_dafny.Zero, _dafny.Zero)
		(_this)._chains = chains
		(_this)._logsDBs = _1_logsDBs
	}
}
func (_this *Interop) SealedBlockForVerifiedAtTimestamp(chainID _dafny.Int, ts _dafny.Int) m_Types.Option {
	{
		var _0_verifiedHeads _dafny.Map = (((_this).VerifiedDB()).Get(ts)).Dtor_l2Heads()
		_ = _0_verifiedHeads
		return (((_this).LogsDBs()).Get(chainID).(*m_LogsDB.LogsDB)).FindSealedBlock(((_0_verifiedHeads).Get(chainID).(m_Types.BlockID)).Dtor_number())
	}
}
func (_this *Interop) NextTimestamp() _dafny.Int {
	{
		var _source0 m_Types.Option = ((_this).VerifiedDB()).LastTimestamp()
		_ = _source0
		{
			if _source0.Is_None() {
				return (_this).ActivationTimestamp()
			}
		}
		{
			var _0_ts _dafny.Int = _source0.Get_().(m_Types.Option_Some).Value.(_dafny.Int)
			_ = _0_ts
			return (_0_ts).Plus(_dafny.One)
		}
	}
}
func (_this *Interop) ValidNonGhost() bool {
	{
		return ((((((((((((_this).ActivationTimestamp()).Cmp(m_Types.Companion_Default___.ACTIVATION__TIMESTAMP()) == 0) && (((_this).MessageExpiryWindow()).Cmp(m_Types.Companion_Default___.MESSAGE__EXPIRY__WINDOW()) == 0)) && ((((_this).Chains()).Keys()).Equals(m_Types.Companion_Default___.CHAIN__IDS()))) && ((((_this).LogsDBs()).Keys()).Equals(m_Types.Companion_Default___.CHAIN__IDS()))) && (_dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_k1 _dafny.Int
			_0_k1 = interface{}(_forall_var_0).(_dafny.Int)
			return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_1 _dafny.Int) bool {
				var _1_k2 _dafny.Int
				_1_k2 = interface{}(_forall_var_1).(_dafny.Int)
				return !((((((_this).LogsDBs()).Keys()).Contains(_0_k1)) && ((((_this).LogsDBs()).Keys()).Contains(_1_k2))) && ((_0_k1).Cmp(_1_k2) != 0)) || ((((_this).LogsDBs()).Get(_0_k1).(*m_LogsDB.LogsDB)) != (((_this).LogsDBs()).Get(_1_k2).(*m_LogsDB.LogsDB)) /* dircomp */)
			})
		}))) && (((_this).VerifiedDB()).Valid())) && (!(((_this).VerifiedDB().Go__lastTimestamp).Is_Some()) || (((_this).VerifiedDB().Db).Contains((_this).ActivationTimestamp())))) && (_dafny.Quantifier(((_this).VerifiedDB().Db).Keys().Elements(), true, func(_forall_var_2 _dafny.Int) bool {
			var _2_ts _dafny.Int
			_2_ts = interface{}(_forall_var_2).(_dafny.Int)
			return !(((_this).VerifiedDB().Db).Contains(_2_ts)) || (((_this).ActivationTimestamp()).Cmp(_2_ts) <= 0)
		}))) && (_dafny.Quantifier(((_this).VerifiedDB().Db).Keys().Elements(), true, func(_forall_var_3 _dafny.Int) bool {
			var _3_ts _dafny.Int
			_3_ts = interface{}(_forall_var_3).(_dafny.Int)
			return !(((_this).VerifiedDB().Db).Contains(_3_ts)) || ((((((_this).VerifiedDB().Db).Get(_3_ts).(m_Types.VerifiedResult)).Dtor_l2Heads()).Keys()).Equals(m_Types.Companion_Default___.CHAIN__IDS()))
		}))) && (!(((_this).VerifiedDB().PendingTransition).Is_Some()) || (m_Types.Companion_Default___.ValidPendingTransition((((_this).VerifiedDB()).GetPendingTransition()).Dtor_value().(m_Types.PendingTransition))))) && ((_this).AllVerifiedHeadsBoundedByTimestamp())
	}
}
func (_this *Interop) DBsInSyncUpTo(chainID _dafny.Int, upperTS _dafny.Int) bool {
	{
		return _dafny.Quantifier(_dafny.IntegerRange((_this).ActivationTimestamp(), (upperTS).Plus(_dafny.One)), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_t _dafny.Int
			_0_t = interface{}(_forall_var_0).(_dafny.Int)
			return !((((_this).ActivationTimestamp()).Cmp(_0_t) <= 0) && ((_0_t).Cmp(upperTS) <= 0)) || ((((((_this).VerifiedDB()).Has(_0_t)) && (((((_this).VerifiedDB()).Get(_0_t)).Dtor_l2Heads()).Contains(chainID))) && (((_this).SealedBlockForVerifiedAtTimestamp(chainID, _0_t)).Is_Some())) && (((((_this).SealedBlockForVerifiedAtTimestamp(chainID, _0_t)).Dtor_value().(m_Types.BlockSeal)).Dtor_id()).Equals(((((_this).VerifiedDB()).Get(_0_t)).Dtor_l2Heads()).Get(chainID).(m_Types.BlockID))))
		})
	}
}
func (_this *Interop) DBsInSync(chainID _dafny.Int) bool {
	{
		var _source0 m_Types.Option = ((_this).VerifiedDB()).LastTimestamp()
		_ = _source0
		{
			if _source0.Is_None() {
				return ((((_this).LogsDBs()).Get(chainID).(*m_LogsDB.LogsDB)).LatestSealedBlock()).Equals(m_Types.Companion_Option_.Create_None_())
			}
		}
		{
			var _0_ts _dafny.Int = _source0.Get_().(m_Types.Option_Some).Value.(_dafny.Int)
			_ = _0_ts
			var _1_l2Heads _dafny.Map = (((_this).VerifiedDB()).Get(_0_ts)).Dtor_l2Heads()
			_ = _1_l2Heads
			return (((_1_l2Heads).Contains(chainID)) && (((((_this).LogsDBs()).Get(chainID).(*m_LogsDB.LogsDB)).LatestSealedBlock()).Equals(m_Types.Companion_Option_.Create_Some_((_1_l2Heads).Get(chainID).(m_Types.BlockID))))) && ((_this).DBsInSyncUpTo(chainID, _0_ts))
		}
	}
}
func (_this *Interop) AllDBsInSyncUpTo(upper _dafny.Int) bool {
	{
		return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_k _dafny.Int
			_0_k = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_k) {
				return !((((_this).LogsDBs()).Keys()).Contains(_0_k)) || ((_this).DBsInSyncUpTo(_0_k, upper))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) AllDBsInSync() bool {
	{
		return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_k _dafny.Int
			_0_k = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_k) {
				return !((((_this).LogsDBs()).Keys()).Contains(_0_k)) || ((_this).DBsInSync(_0_k))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) PlanConsistentWithVerified(plan m_Types.RewindPlan) bool {
	{
		var _pat_let_tv0 = plan
		_ = _pat_let_tv0
		var _pat_let_tv1 = plan
		_ = _pat_let_tv1
		var _pat_let_tv2 = plan
		_ = _pat_let_tv2
		return !(((plan).Dtor_resetAllChainsTo()).Is_Some()) || (func(_pat_let3_0 _dafny.Int) bool {
			return func(_0_ts _dafny.Int) bool {
				return ((((_this).VerifiedDB()).Has(_0_ts)) && (_dafny.Quantifier(_dafny.IntegerRange(_dafny.Zero, (_pat_let_tv0).Dtor_rewindAtOrAfter()), true, func(_forall_var_0 _dafny.Int) bool {
					var _1_t _dafny.Int
					_1_t = interface{}(_forall_var_0).(_dafny.Int)
					return !((((_1_t).Sign() != -1) && ((_1_t).Cmp((_pat_let_tv1).Dtor_rewindAtOrAfter()) < 0)) && (((_this).VerifiedDB()).Has(_1_t))) || ((_1_t).Cmp(_0_ts) <= 0)
				}))) && (((_pat_let_tv2).Dtor_targetHeads()).Equals((((_this).VerifiedDB()).Get(_0_ts)).Dtor_l2Heads()))
			}(_pat_let3_0)
		}(((plan).Dtor_resetAllChainsTo()).Dtor_value().(_dafny.Int)))
	}
}
func (_this *Interop) PlanConsistentWithLogs(plan m_Types.RewindPlan, chainID _dafny.Int) bool {
	{
		var _pat_let_tv0 = plan
		_ = _pat_let_tv0
		var _pat_let_tv1 = chainID
		_ = _pat_let_tv1
		return !(((plan).Dtor_resetAllChainsTo()).Is_Some()) || (func(_pat_let4_0 m_Types.Option) bool {
			return func(_0_sealedBlock m_Types.Option) bool {
				return ((_0_sealedBlock).Is_Some()) && ((((_0_sealedBlock).Dtor_value().(m_Types.BlockSeal)).Dtor_id()).Equals(((_pat_let_tv0).Dtor_targetHeads()).Get(_pat_let_tv1).(m_Types.BlockID)))
			}(_pat_let4_0)
		}((((_this).LogsDBs()).Get(chainID).(*m_LogsDB.LogsDB)).FindSealedBlock((((plan).Dtor_targetHeads()).Get(chainID).(m_Types.BlockID)).Dtor_number())))
	}
}
func (_this *Interop) PlanConsistentWithAllLogs(plan m_Types.RewindPlan) bool {
	{
		return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_chainID _dafny.Int
			_0_chainID = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_chainID) {
				return !((((_this).LogsDBs()).Keys()).Contains(_0_chainID)) || ((_this).PlanConsistentWithLogs(plan, _0_chainID))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) RewoundVerifiedDB(plan m_Types.RewindPlan) bool {
	{
		return (((((_this).VerifiedDB().PendingTransition).Is_Some()) && (((((_this).VerifiedDB().PendingTransition).Dtor_value().(m_Types.PendingTransition)).Dtor_decision()).Equals(m_Types.Companion_Decision_.Create_Rewind_()))) && (((((_this).VerifiedDB().PendingTransition).Dtor_value().(m_Types.PendingTransition)).Dtor_rewind()).Equals(m_Types.Companion_Option_.Create_Some_(plan)))) && (func() bool {
			var _source0 m_Types.Option = (plan).Dtor_resetAllChainsTo()
			_ = _source0
			{
				if _source0.Is_None() {
					return (((_this).VerifiedDB().Db).Cardinality()).Sign() == 0
				}
			}
			{
				var _0_ts _dafny.Int = _source0.Get_().(m_Types.Option_Some).Value.(_dafny.Int)
				_ = _0_ts
				return ((((_this).VerifiedDB()).LastTimestamp()).Equals(m_Types.Companion_Option_.Create_Some_(_0_ts))) && (((plan).Dtor_targetHeads()).Equals((((_this).VerifiedDB()).Get(_0_ts)).Dtor_l2Heads()))
			}
		}())
	}
}
func (_this *Interop) RewoundLogsDB(plan m_Types.RewindPlan, chainID _dafny.Int) bool {
	{
		var _source0 m_Types.Option = (plan).Dtor_resetAllChainsTo()
		_ = _source0
		{
			if _source0.Is_None() {
				return ((((_this).LogsDBs()).Get(chainID).(*m_LogsDB.LogsDB)).LatestSealedBlock()).Equals(m_Types.Companion_Option_.Create_None_())
			}
		}
		{
			return ((((_this).LogsDBs()).Get(chainID).(*m_LogsDB.LogsDB)).LatestSealedBlock()).Equals(m_Types.Companion_Option_.Create_Some_(((plan).Dtor_targetHeads()).Get(chainID).(m_Types.BlockID)))
		}
	}
}
func (_this *Interop) RewoundAllLogsDB(plan m_Types.RewindPlan) bool {
	{
		return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_chainID _dafny.Int
			_0_chainID = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_chainID) {
				return !((((_this).LogsDBs()).Keys()).Contains(_0_chainID)) || ((_this).RewoundLogsDB(plan, _0_chainID))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) TransitionConsistentWithVerified(pending m_Types.PendingTransition) bool {
	{
		var _source0 m_Types.Decision = (pending).Dtor_decision()
		_ = _source0
		{
			if _source0.Is_Rewind() {
				return (_this).PlanConsistentWithVerified(((pending).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan))
			}
		}
		{
			if _source0.Is_Invalidate() {
				return true
			}
		}
		{
			var _0_newTimestamp _dafny.Int = (((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_timestamp()
			_ = _0_newTimestamp
			var _1_newL2Heads _dafny.Map = (((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_l2Heads()
			_ = _1_newL2Heads
			return (_this).AdvancesVerifiedDB(_0_newTimestamp, _1_newL2Heads)
		}
	}
}
func (_this *Interop) TransitionConsistentWithLogs(pending m_Types.PendingTransition) bool {
	{
		var _source0 m_Types.Decision = (pending).Dtor_decision()
		_ = _source0
		{
			if _source0.Is_Rewind() {
				return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
					var _0_k _dafny.Int
					_0_k = interface{}(_forall_var_0).(_dafny.Int)
					if m__System.Companion_Nat_.Is_(_0_k) {
						return !(((((_this).LogsDBs()).Keys()).Contains(_0_k)) && (((((pending).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan)).Dtor_resetAllChainsTo()).Is_Some())) || ((((((pending).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan)).Dtor_targetHeads()).Contains(_0_k)) && ((_this).PlanConsistentWithLogs(((pending).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan), _0_k)))
					} else {
						return true
					}
				})
			}
		}
		{
			if _source0.Is_Invalidate() {
				return true
			}
		}
		{
			var _1_newTimestamp _dafny.Int = (((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_timestamp()
			_ = _1_newTimestamp
			var _2_newL2Heads _dafny.Map = (((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_l2Heads()
			_ = _2_newL2Heads
			return (((_2_newL2Heads).Keys()).Equals(((_this).LogsDBs()).Keys())) && ((_this).AdvancesAllLogsDBs(_1_newTimestamp, _2_newL2Heads))
		}
	}
}
func (_this *Interop) TransitionConsistentWithChainState(pending m_Types.PendingTransition) bool {
	{
		return !(((pending).Dtor_decision()).Is_Advance()) || (((_this).BlocksExistedOnChain((((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_l2Heads())) && ((_this).FrontierBlocksConsistentWithTimestamp((((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_timestamp(), (((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_l2Heads())))
	}
}
func (_this *Interop) PendingTransitionIsConsistent() bool {
	{
		var _source0 m_Types.Option = ((_this).VerifiedDB()).GetPendingTransition()
		_ = _source0
		{
			if _source0.Is_None() {
				return (_this).AllDBsInSync()
			}
		}
		{
			var _0_p m_Types.PendingTransition = _source0.Get_().(m_Types.Option_Some).Value.(m_Types.PendingTransition)
			_ = _0_p
			return ((((_this).TransitionConsistentWithVerified(_0_p)) && ((_this).TransitionConsistentWithLogs(_0_p))) && ((_this).TransitionConsistentWithChainState(_0_p))) && (func() bool {
				var _source1 m_Types.Decision = (_0_p).Dtor_decision()
				_ = _source1
				{
					if _source1.Is_Rewind() {
						return !(((((_0_p).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan)).Dtor_resetAllChainsTo()).Is_Some()) || ((_this).AllDBsInSyncUpTo(((((_0_p).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan)).Dtor_resetAllChainsTo()).Dtor_value().(_dafny.Int)))
					}
				}
				{
					if _source1.Is_Invalidate() {
						return (_this).AllDBsInSync()
					}
				}
				{
					return !((((_this).VerifiedDB()).LastTimestamp()).Is_Some()) || ((_this).AllDBsInSyncUpTo((((_this).VerifiedDB()).LastTimestamp()).Dtor_value().(_dafny.Int)))
				}
			}())
		}
	}
}
func (_this *Interop) OutputConsistentWithVerified(output m_Types.StepOutput, obs m_Types.RoundObservation) bool {
	{
		var _source0 m_Types.StepOutput = output
		_ = _source0
		{
			if _source0.Is_WaitOutput() {
				return true
			}
		}
		{
			if _source0.Is_RewindOutput() {
				return !(((_this).ActivationTimestamp()).Cmp(((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)) < 0) || (((_this).VerifiedDB().Db).Contains((((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)).Minus(_dafny.One)))
			}
		}
		{
			if _source0.Is_AdvanceOutput() {
				var _0_result m_Types.Result = _source0.Get_().(m_Types.StepOutput_AdvanceOutput).Result
				_ = _0_result
				return (_this).AdvancesVerifiedDB((_0_result).Dtor_timestamp(), (_0_result).Dtor_l2Heads())
			}
		}
		{
			var _1_result m_Types.Result = _source0.Get_().(m_Types.StepOutput_InvalidateOutput).Result
			_ = _1_result
			return ((_1_result).Dtor_timestamp()).Cmp((_this).NextTimestamp()) == 0
		}
	}
}
func (_this *Interop) OutputConsistentWithLogs(output m_Types.StepOutput, obs m_Types.RoundObservation) bool {
	{
		var _pat_let_tv0 = obs
		_ = _pat_let_tv0
		var _source0 m_Types.StepOutput = output
		_ = _source0
		{
			if _source0.Is_WaitOutput() {
				return true
			}
		}
		{
			if _source0.Is_RewindOutput() {
				return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
					var _0_k _dafny.Int
					_0_k = interface{}(_forall_var_0).(_dafny.Int)
					if m__System.Companion_Nat_.Is_(_0_k) {
						return !(((((_this).LogsDBs()).Keys()).Contains(_0_k)) && (((_this).ActivationTimestamp()).Cmp(((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)) < 0)) || (func(_pat_let5_0 m_Types.Option) bool {
							return func(_1_sealedBlock m_Types.Option) bool {
								return ((_1_sealedBlock).Is_Some()) && ((((_1_sealedBlock).Dtor_value().(m_Types.BlockSeal)).Dtor_id()).Equals(((((_this).VerifiedDB()).Get((((_pat_let_tv0).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)).Minus(_dafny.One))).Dtor_l2Heads()).Get(_0_k).(m_Types.BlockID)))
							}(_pat_let5_0)
						}((_this).SealedBlockForVerifiedAtTimestamp(_0_k, (((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)).Minus(_dafny.One))))
					} else {
						return true
					}
				})
			}
		}
		{
			if _source0.Is_AdvanceOutput() {
				var _2_result m_Types.Result = _source0.Get_().(m_Types.StepOutput_AdvanceOutput).Result
				_ = _2_result
				return (_this).AdvancesAllLogsDBs((_2_result).Dtor_timestamp(), (_2_result).Dtor_l2Heads())
			}
		}
		{
			return true
		}
	}
}
func (_this *Interop) OutputConsistentWithChainState(output m_Types.StepOutput, obs m_Types.RoundObservation) bool {
	{
		return !((output).Is_AdvanceOutput()) || (((((((output).Dtor_result()).Dtor_l2Heads()).Keys()).Equals(m_Types.Companion_Default___.CHAIN__IDS())) && ((_this).BlocksExistedOnChain(((output).Dtor_result()).Dtor_l2Heads()))) && ((_this).FrontierBlocksConsistentWithTimestamp(((output).Dtor_result()).Dtor_timestamp(), ((output).Dtor_result()).Dtor_l2Heads())))
	}
}
func (_this *Interop) ObservationConsistentWithVerified(obs m_Types.RoundObservation) bool {
	{
		return (((((obs).Dtor_lastVerifiedTS()).Equals((_this).VerifiedDB().Go__lastTimestamp)) && (((obs).Dtor_nextTimestamp()).Cmp((_this).NextTimestamp()) == 0)) && (!((!((obs).Dtor_l1Consistent())) && (((_this).ActivationTimestamp()).Cmp(((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)) < 0)) || (((_this).VerifiedDB().Db).Contains((((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)).Minus(_dafny.One))))) && (!((((obs).Dtor_chainsReady()) && ((obs).Dtor_l2sConsistent())) && ((obs).Dtor_l1Consistent())) || ((_this).AdvancesVerifiedDB((obs).Dtor_nextTimestamp(), (obs).Dtor_blocksAtTS())))
	}
}
func (_this *Interop) ObservationConsistentWithLogs(obs m_Types.RoundObservation) bool {
	{
		var _pat_let_tv0 = obs
		_ = _pat_let_tv0
		return (!((!((obs).Dtor_l1Consistent())) && ((((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)).Cmp((_this).ActivationTimestamp()) > 0)) || (_dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_k _dafny.Int
			_0_k = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_k) {
				return !((((_this).LogsDBs()).Keys()).Contains(_0_k)) || (func(_pat_let6_0 m_Types.Option) bool {
					return func(_1_sealedBlock m_Types.Option) bool {
						return ((_1_sealedBlock).Is_Some()) && ((((_1_sealedBlock).Dtor_value().(m_Types.BlockSeal)).Dtor_id()).Equals(((((_this).VerifiedDB()).Get((((_pat_let_tv0).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)).Minus(_dafny.One))).Dtor_l2Heads()).Get(_0_k).(m_Types.BlockID)))
					}(_pat_let6_0)
				}((_this).SealedBlockForVerifiedAtTimestamp(_0_k, (((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)).Minus(_dafny.One))))
			} else {
				return true
			}
		}))) && (!((((obs).Dtor_chainsReady()) && ((obs).Dtor_l2sConsistent())) && ((obs).Dtor_l1Consistent())) || ((_this).AdvancesAllLogsDBs((obs).Dtor_nextTimestamp(), (obs).Dtor_blocksAtTS())))
	}
}
func (_this *Interop) ObservationConsistentWithChainState(obs m_Types.RoundObservation) bool {
	{
		return !((((obs).Dtor_blocksAtTS()).Cardinality()).Sign() == 1) || ((((((obs).Dtor_blocksAtTS()).Keys()).Equals(m_Types.Companion_Default___.CHAIN__IDS())) && ((_this).BlocksExistedOnChain((obs).Dtor_blocksAtTS()))) && ((_this).FrontierBlocksConsistentWithTimestamp((obs).Dtor_nextTimestamp(), (obs).Dtor_blocksAtTS())))
	}
}
func (_this *Interop) AdvancesVerifiedDB(ts _dafny.Int, blocksAtTS _dafny.Map) bool {
	{
		var _pat_let_tv0 = blocksAtTS
		_ = _pat_let_tv0
		var _pat_let_tv1 = blocksAtTS
		_ = _pat_let_tv1
		var _pat_let_tv2 = blocksAtTS
		_ = _pat_let_tv2
		var _pat_let_tv3 = blocksAtTS
		_ = _pat_let_tv3
		var _pat_let_tv4 = blocksAtTS
		_ = _pat_let_tv4
		var _source0 m_Types.Option = ((_this).VerifiedDB()).LastTimestamp()
		_ = _source0
		{
			if _source0.Is_None() {
				return (ts).Cmp((_this).ActivationTimestamp()) == 0
			}
		}
		{
			var _0_lastTS _dafny.Int = _source0.Get_().(m_Types.Option_Some).Value.(_dafny.Int)
			_ = _0_lastTS
			return ((ts).Cmp((_0_lastTS).Plus(_dafny.One)) == 0) && (func(_pat_let7_0 m_Types.VerifiedResult) bool {
				return func(_1_lastVerifiedResult m_Types.VerifiedResult) bool {
					return (((_pat_let_tv0).Keys()).Equals(((_1_lastVerifiedResult).Dtor_l2Heads()).Keys())) && (_dafny.Quantifier(((_pat_let_tv1).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
						var _2_chainID _dafny.Int
						_2_chainID = interface{}(_forall_var_0).(_dafny.Int)
						return !(((_pat_let_tv2).Keys()).Contains(_2_chainID)) || (func(_pat_let8_0 m_Types.BlockID) bool {
							return func(_3_lastBlock m_Types.BlockID) bool {
								return (((_3_lastBlock).Dtor_number()).Cmp(((_pat_let_tv3).Get(_2_chainID).(m_Types.BlockID)).Dtor_number()) <= 0) && ((((_pat_let_tv4).Get(_2_chainID).(m_Types.BlockID)).Dtor_number()).Cmp(((_3_lastBlock).Dtor_number()).Plus(_dafny.One)) <= 0)
							}(_pat_let8_0)
						}(((_1_lastVerifiedResult).Dtor_l2Heads()).Get(_2_chainID).(m_Types.BlockID)))
					}))
				}(_pat_let7_0)
			}(((_this).VerifiedDB()).Get(_0_lastTS)))
		}
	}
}
func (_this *Interop) AdvancesLogsDB(ts _dafny.Int, chainID _dafny.Int, newBlock m_Types.BlockID) bool {
	{
		var _source0 m_Types.Option = (((_this).LogsDBs()).Get(chainID).(*m_LogsDB.LogsDB)).LatestSealedBlock()
		_ = _source0
		{
			if _source0.Is_None() {
				return (ts).Cmp((_this).ActivationTimestamp()) == 0
			}
		}
		{
			var _0_latestBlock m_Types.BlockID = _source0.Get_().(m_Types.Option_Some).Value.(m_Types.BlockID)
			_ = _0_latestBlock
			return ((((_0_latestBlock).Dtor_number()).Cmp((newBlock).Dtor_number()) <= 0) && (((newBlock).Dtor_number()).Cmp(((_0_latestBlock).Dtor_number()).Plus(_dafny.One)) <= 0)) && (!(((_0_latestBlock).Dtor_number()).Cmp((newBlock).Dtor_number()) == 0) || ((_0_latestBlock).Equals(newBlock)))
		}
	}
}
func (_this *Interop) AdvancesAllLogsDBs(ts _dafny.Int, blocksAtTS _dafny.Map) bool {
	{
		return _dafny.Quantifier((((_this).LogsDBs()).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_chainID _dafny.Int
			_0_chainID = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_chainID) {
				return !((((_this).LogsDBs()).Keys()).Contains(_0_chainID)) || ((_this).AdvancesLogsDB(ts, _0_chainID, (blocksAtTS).Get(_0_chainID).(m_Types.BlockID)))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) ValidExecutingMessage(execTimestamp _dafny.Int, execChain _dafny.Int, execMsg m_Types.ExecutingMessage) bool {
	{
		var _pat_let_tv0 = execChain
		_ = _pat_let_tv0
		var _pat_let_tv1 = execTimestamp
		_ = _pat_let_tv1
		var _pat_let_tv2 = execTimestamp
		_ = _pat_let_tv2
		var _pat_let_tv3 = execTimestamp
		_ = _pat_let_tv3
		var _0_initChain _dafny.Int = (execMsg).Dtor_chainID()
		_ = _0_initChain
		var _1_initTimestamp _dafny.Int = (execMsg).Dtor_timestamp()
		_ = _1_initTimestamp
		return ((((_this).Chains()).Keys()).Contains(_0_initChain)) && (func(_pat_let9_0 _dafny.Int) bool {
			return func(_2_initBlockTime _dafny.Int) bool {
				return func(_pat_let10_0 _dafny.Int) bool {
					return func(_3_execBlockTime _dafny.Int) bool {
						return (((((_this).ActivationTimestamp()).Plus(_3_execBlockTime)).Cmp(_pat_let_tv1) <= 0) && ((((_this).ActivationTimestamp()).Plus(_2_initBlockTime)).Cmp(_1_initTimestamp) <= 0)) && (((_1_initTimestamp).Cmp(_pat_let_tv2) <= 0) && ((_pat_let_tv3).Cmp((_1_initTimestamp).Plus((_this).MessageExpiryWindow())) <= 0))
					}(_pat_let10_0)
				}((((_this).Chains()).Get(_pat_let_tv0).(*m_ChainContainer.ChainContainer)).BlockTime())
			}(_pat_let9_0)
		}((((_this).Chains()).Get(_0_initChain).(*m_ChainContainer.ChainContainer)).BlockTime()))
	}
}
func (_this *Interop) IsCorrectFrontierView(view *FrontierView, blocksAtTS _dafny.Map) bool {
	{
		return _dafny.Quantifier((m_Types.Companion_Default___.CHAIN__IDS()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_chainID _dafny.Int
			_0_chainID = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_chainID) {
				return !((m_Types.Companion_Default___.CHAIN__IDS()).Contains(_0_chainID)) || (((((((_this).Chains()).Get(_0_chainID).(*m_ChainContainer.ChainContainer)).BlockInfo((blocksAtTS).Get(_0_chainID).(m_Types.BlockID))).Is_Some()) && (((view).BlockInfo(_0_chainID)).Equals(((((_this).Chains()).Get(_0_chainID).(*m_ChainContainer.ChainContainer)).BlockInfo((blocksAtTS).Get(_0_chainID).(m_Types.BlockID))).Dtor_value().(m_Types.BlockInfo)))) && (((view).BlockLogs(_0_chainID)).Equals(((((_this).Chains()).Get(_0_chainID).(*m_ChainContainer.ChainContainer)).BlockLogs((blocksAtTS).Get(_0_chainID).(m_Types.BlockID))).Dtor_value().(m_Types.BlockLogs))))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) InitMsgInFrontierView(execMsg m_Types.ExecutingMessage, view *FrontierView) bool {
	{
		var _0_query m_Types.ContainsQuery = m_Types.Companion_ContainsQuery_.Create_ContainsQuery_((execMsg).Dtor_blockNum(), (execMsg).Dtor_logIdx(), (execMsg).Dtor_timestamp(), (execMsg).Dtor_checksum())
		_ = _0_query
		return (view).Contains((execMsg).Dtor_chainID(), _0_query)
	}
}
func (_this *Interop) InitMsgInFrontier(execMsg m_Types.ExecutingMessage, blocksAtTS _dafny.Map) bool {
	{
		var _pat_let_tv0 = execMsg
		_ = _pat_let_tv0
		var _pat_let_tv1 = execMsg
		_ = _pat_let_tv1
		var _pat_let_tv2 = execMsg
		_ = _pat_let_tv2
		var _pat_let_tv3 = execMsg
		_ = _pat_let_tv3
		var _pat_let_tv4 = execMsg
		_ = _pat_let_tv4
		var _0_initBlock m_Types.BlockID = (blocksAtTS).Get((execMsg).Dtor_chainID()).(m_Types.BlockID)
		_ = _0_initBlock
		return (((_0_initBlock).Dtor_number()).Cmp((execMsg).Dtor_blockNum()) == 0) && (func(_pat_let11_0 m_Types.Option) bool {
			return func(_1_initBlockInfo m_Types.Option) bool {
				return (((_1_initBlockInfo).Is_Some()) && ((((_1_initBlockInfo).Dtor_value().(m_Types.BlockInfo)).Dtor_timestamp()).Cmp((_pat_let_tv0).Dtor_timestamp()) == 0)) && (func(_pat_let12_0 m_Types.Option) bool {
					return func(_2_initBlockLogs m_Types.Option) bool {
						return (((_pat_let_tv2).Dtor_logIdx()).Cmp(_dafny.IntOfUint32((((_2_initBlockLogs).Dtor_value().(m_Types.BlockLogs)).Dtor_fullLogs()).Cardinality())) < 0) && (func(_pat_let13_0 m_Types.Log) bool {
							return func(_3_initMsg m_Types.Log) bool {
								return ((_3_initMsg).Dtor_checksum()).Cmp((_pat_let_tv4).Dtor_checksum()) == 0
							}(_pat_let13_0)
						}((((_2_initBlockLogs).Dtor_value().(m_Types.BlockLogs)).Dtor_fullLogs()).Select(((_pat_let_tv3).Dtor_logIdx()).Uint32()).(m_Types.Log)))
					}(_pat_let12_0)
				}((((_this).Chains()).Get((_pat_let_tv1).Dtor_chainID()).(*m_ChainContainer.ChainContainer)).BlockLogs(_0_initBlock)))
			}(_pat_let11_0)
		}((((_this).Chains()).Get((execMsg).Dtor_chainID()).(*m_ChainContainer.ChainContainer)).BlockInfo(_0_initBlock)))
	}
}
func (_this *Interop) InitMsgInLogsDB(execMsg m_Types.ExecutingMessage) bool {
	{
		var _0_query m_Types.ContainsQuery = m_Types.Companion_ContainsQuery_.Create_ContainsQuery_((execMsg).Dtor_blockNum(), (execMsg).Dtor_logIdx(), (execMsg).Dtor_timestamp(), (execMsg).Dtor_checksum())
		_ = _0_query
		return (((_this).LogsDBs()).Get((execMsg).Dtor_chainID()).(*m_LogsDB.LogsDB)).Contains(_0_query)
	}
}
func (_this *Interop) BlockIsCrossValid(ts _dafny.Int, chainID _dafny.Int, blockID m_Types.BlockID) bool {
	{
		var _0_logs m_Types.BlockLogs = ((((_this).Chains()).Get(chainID).(*m_ChainContainer.ChainContainer)).BlockLogs(blockID)).Dtor_value().(m_Types.BlockLogs)
		_ = _0_logs
		return _dafny.Quantifier((((_0_logs).Dtor_execMsgs()).Values()).Elements(), true, func(_forall_var_0 m_Types.ExecutingMessage) bool {
			var _1_execMsg m_Types.ExecutingMessage
			_1_execMsg = interface{}(_forall_var_0).(m_Types.ExecutingMessage)
			return !((((_0_logs).Dtor_execMsgs()).Values()).Contains(_1_execMsg)) || ((_this).ValidExecutingMessage(ts, chainID, _1_execMsg))
		})
	}
}
func (_this *Interop) AllInitMsgsPresent(chainID _dafny.Int, blockID m_Types.BlockID, blocksAtTS _dafny.Map) bool {
	{
		var _0_logs m_Types.BlockLogs = ((((_this).Chains()).Get(chainID).(*m_ChainContainer.ChainContainer)).BlockLogs(blockID)).Dtor_value().(m_Types.BlockLogs)
		_ = _0_logs
		return _dafny.Quantifier((((_0_logs).Dtor_execMsgs()).Values()).Elements(), true, func(_forall_var_0 m_Types.ExecutingMessage) bool {
			var _1_execMsg m_Types.ExecutingMessage
			_1_execMsg = interface{}(_forall_var_0).(m_Types.ExecutingMessage)
			return !((((_0_logs).Dtor_execMsgs()).Values()).Contains(_1_execMsg)) || (((m_Types.Companion_Default___.CHAIN__IDS()).Contains((_1_execMsg).Dtor_chainID())) && (((_this).InitMsgInFrontier(_1_execMsg, blocksAtTS)) || ((_this).InitMsgInLogsDB(_1_execMsg))))
		})
	}
}
func (_this *Interop) AllInitMsgsInLogsDB(chainID _dafny.Int, blockID m_Types.BlockID) bool {
	{
		var _0_logs m_Types.BlockLogs = ((((_this).Chains()).Get(chainID).(*m_ChainContainer.ChainContainer)).BlockLogs(blockID)).Dtor_value().(m_Types.BlockLogs)
		_ = _0_logs
		return _dafny.Quantifier((((_0_logs).Dtor_execMsgs()).Values()).Elements(), true, func(_forall_var_0 m_Types.ExecutingMessage) bool {
			var _1_execMsg m_Types.ExecutingMessage
			_1_execMsg = interface{}(_forall_var_0).(m_Types.ExecutingMessage)
			return !((((_0_logs).Dtor_execMsgs()).Values()).Contains(_1_execMsg)) || (((((_this).LogsDBs()).Keys()).Contains((_1_execMsg).Dtor_chainID())) && ((_this).InitMsgInLogsDB(_1_execMsg)))
		})
	}
}
func (_this *Interop) ResultIsCrossValid(result m_Types.Result) bool {
	{
		var _pat_let_tv0 = result
		_ = _pat_let_tv0
		return _dafny.Quantifier((m_Types.Companion_Default___.CHAIN__IDS()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_chainID _dafny.Int
			_0_chainID = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_chainID) {
				return !((m_Types.Companion_Default___.CHAIN__IDS()).Contains(_0_chainID)) || (func(_pat_let14_0 m_Types.BlockID) bool {
					return func(_1_blockID m_Types.BlockID) bool {
						return func(_pat_let15_0 m_Types.Option) bool {
							return func(_2_blockInfo m_Types.Option) bool {
								return ((_2_blockInfo).Is_Some()) && (func(_pat_let16_0 _dafny.Int) bool {
									return func(_3_ts _dafny.Int) bool {
										return ((_this).BlockIsCrossValid(_3_ts, _0_chainID, _1_blockID)) && ((_this).AllInitMsgsPresent(_0_chainID, _1_blockID, (_pat_let_tv0).Dtor_l2Heads()))
									}(_pat_let16_0)
								}(((_2_blockInfo).Dtor_value().(m_Types.BlockInfo)).Dtor_timestamp()))
							}(_pat_let15_0)
						}((((_this).Chains()).Get(_0_chainID).(*m_ChainContainer.ChainContainer)).BlockInfo(_1_blockID))
					}(_pat_let14_0)
				}(((result).Dtor_l2Heads()).Get(_0_chainID).(m_Types.BlockID)))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) BlockExistedOnChain(chainID _dafny.Int, blockID m_Types.BlockID) bool {
	{
		return ((((_this).Chains()).Get(chainID).(*m_ChainContainer.ChainContainer)).BlockLogs(blockID)).Is_Some()
	}
}
func (_this *Interop) BlocksExistedOnChain(blocksAtTS _dafny.Map) bool {
	{
		return _dafny.Quantifier(((blocksAtTS).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_chainID _dafny.Int
			_0_chainID = interface{}(_forall_var_0).(_dafny.Int)
			if m__System.Companion_Nat_.Is_(_0_chainID) {
				return !(((blocksAtTS).Keys()).Contains(_0_chainID)) || ((_this).BlockExistedOnChain(_0_chainID, (blocksAtTS).Get(_0_chainID).(m_Types.BlockID)))
			} else {
				return true
			}
		})
	}
}
func (_this *Interop) TransitionIsCrossValid(pendingTransition m_Types.PendingTransition) bool {
	{
		return !(((pendingTransition).Dtor_decision()).Is_Advance()) || ((_this).ResultIsCrossValid(((pendingTransition).Dtor_result()).Dtor_value().(m_Types.Result)))
	}
}
func (_this *Interop) AllVerifiedCrossValid() bool {
	{
		return !((((_this).VerifiedDB()).LastTimestamp()).Is_Some()) || (_dafny.Quantifier(_dafny.IntegerRange((_this).ActivationTimestamp(), ((((_this).VerifiedDB()).LastTimestamp()).Dtor_value().(_dafny.Int)).Plus(_dafny.One)), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_ts _dafny.Int
			_0_ts = interface{}(_forall_var_0).(_dafny.Int)
			return !((((_this).ActivationTimestamp()).Cmp(_0_ts) <= 0) && ((_0_ts).Cmp((((_this).VerifiedDB()).LastTimestamp()).Dtor_value().(_dafny.Int)) <= 0)) || (func(_pat_let17_0 m_Types.VerifiedResult) bool {
				return func(_1_verified m_Types.VerifiedResult) bool {
					return func(_pat_let18_0 m_Types.Result) bool {
						return func(_2_result m_Types.Result) bool {
							return (_this).ResultIsCrossValid(_2_result)
						}(_pat_let18_0)
					}(m_Types.Companion_Result_.Create_Result_((_1_verified).Dtor_timestamp(), (_1_verified).Dtor_l1Inclusion(), (_1_verified).Dtor_l2Heads(), _dafny.NewMapBuilder().ToMap()))
				}(_pat_let17_0)
			}(((_this).VerifiedDB()).Get(_0_ts)))
		}))
	}
}
func (_this *Interop) VerifiedHeadsBoundedByTimestamp(ts _dafny.Int) bool {
	{
		var _0_result m_Types.VerifiedResult = ((_this).VerifiedDB()).Get(ts)
		_ = _0_result
		return _dafny.Quantifier(((_0_result).Dtor_l2Heads()).Keys().Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _1_chainID _dafny.Int
			_1_chainID = interface{}(_forall_var_0).(_dafny.Int)
			return !(((_0_result).Dtor_l2Heads()).Contains(_1_chainID)) || (((((((_this).Chains()).Get(_1_chainID).(*m_ChainContainer.ChainContainer)).BlockInfo(((_0_result).Dtor_l2Heads()).Get(_1_chainID).(m_Types.BlockID))).Dtor_value().(m_Types.BlockInfo)).Dtor_timestamp()).Cmp(ts) <= 0)
		})
	}
}
func (_this *Interop) AllVerifiedHeadsBoundedByTimestamp() bool {
	{
		var _0_lastTimestamp m_Types.Option = ((_this).VerifiedDB()).LastTimestamp()
		_ = _0_lastTimestamp
		return !((_0_lastTimestamp).Is_Some()) || (_dafny.Quantifier(_dafny.IntegerRange((_this).ActivationTimestamp(), ((((_this).VerifiedDB()).LastTimestamp()).Dtor_value().(_dafny.Int)).Plus(_dafny.One)), true, func(_forall_var_0 _dafny.Int) bool {
			var _1_ts _dafny.Int
			_1_ts = interface{}(_forall_var_0).(_dafny.Int)
			return !((((_this).ActivationTimestamp()).Cmp(_1_ts) <= 0) && ((_1_ts).Cmp((((_this).VerifiedDB()).LastTimestamp()).Dtor_value().(_dafny.Int)) <= 0)) || ((((((_this).VerifiedDB()).Has(_1_ts)) && ((((_this).Chains()).Keys()).Equals(((((_this).VerifiedDB()).Get(_1_ts)).Dtor_l2Heads()).Keys()))) && ((_this).BlocksExistedOnChain((((_this).VerifiedDB()).Get(_1_ts)).Dtor_l2Heads()))) && ((_this).VerifiedHeadsBoundedByTimestamp(_1_ts)))
		}))
	}
}
func (_this *Interop) FrontierBlocksConsistentWithTimestamp(ts _dafny.Int, blocksAtTS _dafny.Map) bool {
	{
		return _dafny.Quantifier(((blocksAtTS).Keys()).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_chainID _dafny.Int
			_0_chainID = interface{}(_forall_var_0).(_dafny.Int)
			return !(((blocksAtTS).Keys()).Contains(_0_chainID)) || (((((((_this).Chains()).Get(_0_chainID).(*m_ChainContainer.ChainContainer)).BlockInfo((blocksAtTS).Get(_0_chainID).(m_Types.BlockID))).Dtor_value().(m_Types.BlockInfo)).Dtor_timestamp()).Cmp(ts) <= 0)
		})
	}
}
func (_this *Interop) ProgressAndRecord() m_Types.Option {
	{
		var madeProgress m_Types.Option = m_Types.Companion_Option_.Default()
		_ = madeProgress
		var _0_pending m_Types.Option
		_ = _0_pending
		_0_pending = ((_this).VerifiedDB()).GetPendingTransition()
		if (_0_pending).Is_Some() {
			var _out0 m_Types.Option
			_ = _out0
			_out0 = (_this).ApplyPendingTransition((_0_pending).Dtor_value().(m_Types.PendingTransition))
			madeProgress = _out0
			return madeProgress
		}
		var _1_output m_Types.StepOutput
		_ = _1_output
		var _2_obs m_Types.RoundObservation
		_ = _2_obs
		var _out1 m_Types.StepOutput
		_ = _out1
		var _out2 m_Types.RoundObservation
		_ = _out2
		_out1, _out2 = (_this).ProgressInterop()
		_1_output = _out1
		_2_obs = _out2
		if (_1_output).Is_WaitOutput() {
			(_this).RefreshCurrentL1OnWait()
			madeProgress = m_Types.Companion_Option_.Create_Some_(false)
			return madeProgress
		}
		var _3_pendingTx m_Types.PendingTransition
		_ = _3_pendingTx
		var _out3 m_Types.PendingTransition
		_ = _out3
		_out3 = (_this).BuildPendingTransition(_1_output, _2_obs)
		_3_pendingTx = _out3
		((_this).VerifiedDB()).SetPendingTransition(_3_pendingTx)
		if (((_3_pendingTx).Dtor_decision()).Equals(m_Types.Companion_Decision_.Create_Rewind_())) && (((((_3_pendingTx).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan)).Dtor_resetAllChainsTo()).Is_Some()) {
			var _4_ts _dafny.Int
			_ = _4_ts
			_4_ts = ((((_3_pendingTx).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan)).Dtor_resetAllChainsTo()).Dtor_value().(_dafny.Int)
		}
		var _out4 m_Types.Option
		_ = _out4
		_out4 = (_this).ApplyPendingTransition(_3_pendingTx)
		madeProgress = _out4
		return madeProgress
	}
}
func (_this *Interop) ProgressInterop() (m_Types.StepOutput, m_Types.RoundObservation) {
	{
		var output m_Types.StepOutput = m_Types.Companion_StepOutput_.Default()
		_ = output
		var obs m_Types.RoundObservation = m_Types.Companion_RoundObservation_.Default()
		_ = obs
		var _out0 m_Types.RoundObservation
		_ = _out0
		_out0 = (_this).ObserveRound()
		obs = _out0
		if !((obs).Dtor_chainsReady()) {
			output = m_Types.Companion_StepOutput_.Create_WaitOutput_()
			return output, obs
		}
		if !((obs).Dtor_l2sConsistent()) {
			output = m_Types.Companion_StepOutput_.Create_WaitOutput_()
			return output, obs
		}
		if !((obs).Dtor_l1Consistent()) {
			output = m_Types.Companion_StepOutput_.Create_RewindOutput_()
			return output, obs
		}
		var _0_result m_Types.Result
		_ = _0_result
		var _out1 m_Types.Result
		_ = _out1
		_out1 = (_this).Verify((obs).Dtor_nextTimestamp(), (obs).Dtor_blocksAtTS(), (obs).Dtor_l1Heads())
		_0_result = _out1
		if ((_0_result).Dtor_l2Heads()).Equals(_dafny.NewMapBuilder().ToMap()) {
			output = m_Types.Companion_StepOutput_.Create_WaitOutput_()
		} else if !((_0_result).Dtor_invalidHeads()).Equals(_dafny.NewMapBuilder().ToMap()) {
			output = m_Types.Companion_StepOutput_.Create_InvalidateOutput_(_0_result)
		} else {
			output = m_Types.Companion_StepOutput_.Create_AdvanceOutput_(_0_result)
		}
		return output, obs
	}
}
func (_this *Interop) ObserveRound() m_Types.RoundObservation {
	{
		var obs m_Types.RoundObservation = m_Types.Companion_RoundObservation_.Default()
		_ = obs
		var _0_lastTS m_Types.Option
		_ = _0_lastTS
		_0_lastTS = ((_this).VerifiedDB()).LastTimestamp()
		if (_0_lastTS).Is_Some() {
			var _1_lastResult m_Types.VerifiedResult
			_ = _1_lastResult
			_1_lastResult = ((_this).VerifiedDB()).Get((_0_lastTS).Dtor_value().(_dafny.Int))
			obs = m_Types.Companion_RoundObservation_.Create_RoundObservation_(m_Types.Companion_Option_.Create_Some_((_0_lastTS).Dtor_value().(_dafny.Int)), m_Types.Companion_Option_.Create_Some_(_1_lastResult), ((_0_lastTS).Dtor_value().(_dafny.Int)).Plus(_dafny.One), false, _dafny.NewMapBuilder().ToMap(), _dafny.NewMapBuilder().ToMap(), true, true)
		} else {
			obs = m_Types.Companion_RoundObservation_.Create_RoundObservation_(m_Types.Companion_Option_.Create_None_(), m_Types.Companion_Option_.Create_None_(), (_this).ActivationTimestamp(), false, _dafny.NewMapBuilder().ToMap(), _dafny.NewMapBuilder().ToMap(), true, true)
		}
		var _2_ready m_Types.Option
		_ = _2_ready
		var _out0 m_Types.Option
		_ = _out0
		_out0 = (_this).CheckChainsReady((obs).Dtor_nextTimestamp())
		_2_ready = _out0
		if (_2_ready).Is_None() {
			return obs
		}
		var _3_dt__update__tmp_h0 m_Types.RoundObservation = obs
		_ = _3_dt__update__tmp_h0
		var _4_dt__update_hl1Heads_h0 _dafny.Map = ((_2_ready).Dtor_value().(m_Types.ChainsReadyResult)).Dtor_l1Heads()
		_ = _4_dt__update_hl1Heads_h0
		var _5_dt__update_hblocksAtTS_h0 _dafny.Map = ((_2_ready).Dtor_value().(m_Types.ChainsReadyResult)).Dtor_blocks()
		_ = _5_dt__update_hblocksAtTS_h0
		var _6_dt__update_hchainsReady_h0 bool = true
		_ = _6_dt__update_hchainsReady_h0
		obs = m_Types.Companion_RoundObservation_.Create_RoundObservation_((_3_dt__update__tmp_h0).Dtor_lastVerifiedTS(), (_3_dt__update__tmp_h0).Dtor_lastVerified(), (_3_dt__update__tmp_h0).Dtor_nextTimestamp(), _6_dt__update_hchainsReady_h0, _5_dt__update_hblocksAtTS_h0, _4_dt__update_hl1Heads_h0, (_3_dt__update__tmp_h0).Dtor_l1Consistent(), (_3_dt__update__tmp_h0).Dtor_l2sConsistent())
		var _7_l1Consistent bool
		_ = _7_l1Consistent
		var _8_l2sConsistent bool
		_ = _8_l2sConsistent
		var _out1 bool
		_ = _out1
		var _out2 bool
		_ = _out2
		_out1, _out2 = (_this).CheckL1Consistent((obs).Dtor_l1Heads(), (obs).Dtor_lastVerified())
		_7_l1Consistent = _out1
		_8_l2sConsistent = _out2
		var _9_dt__update__tmp_h1 m_Types.RoundObservation = obs
		_ = _9_dt__update__tmp_h1
		var _10_dt__update_hl2sConsistent_h0 bool = _8_l2sConsistent
		_ = _10_dt__update_hl2sConsistent_h0
		var _11_dt__update_hl1Consistent_h0 bool = _7_l1Consistent
		_ = _11_dt__update_hl1Consistent_h0
		obs = m_Types.Companion_RoundObservation_.Create_RoundObservation_((_9_dt__update__tmp_h1).Dtor_lastVerifiedTS(), (_9_dt__update__tmp_h1).Dtor_lastVerified(), (_9_dt__update__tmp_h1).Dtor_nextTimestamp(), (_9_dt__update__tmp_h1).Dtor_chainsReady(), (_9_dt__update__tmp_h1).Dtor_blocksAtTS(), (_9_dt__update__tmp_h1).Dtor_l1Heads(), _11_dt__update_hl1Consistent_h0, _10_dt__update_hl2sConsistent_h0)
		return obs
	}
}
func (_this *Interop) CheckChainsReady(ts _dafny.Int) m_Types.Option {
	{
		var result m_Types.Option = m_Types.Companion_Option_.Default()
		_ = result
		var _0_blocks _dafny.Map
		_ = _0_blocks
		_0_blocks = _dafny.NewMapBuilder().ToMap()
		var _1_l1Heads _dafny.Map
		_ = _1_l1Heads
		_1_l1Heads = _dafny.NewMapBuilder().ToMap()
		var _2_chainIDs _dafny.Sequence
		_ = _2_chainIDs
		var _out0 _dafny.Sequence
		_ = _out0
		_out0 = m_Types.Companion_Default___.Enumerate(((_this).Chains()).Keys())
		_2_chainIDs = _out0
		var _hi0 _dafny.Int = _dafny.IntOfUint32((_2_chainIDs).Cardinality())
		_ = _hi0
		for _3_i := _dafny.Zero; _3_i.Cmp(_hi0) < 0; _3_i = _3_i.Plus(_dafny.One) {
			var _4_chainID _dafny.Int
			_ = _4_chainID
			_4_chainID = (_2_chainIDs).Select((_3_i).Uint32()).(_dafny.Int)
			var _5_chainResult m_Types.Option
			_ = _5_chainResult
			var _out1 m_Types.Option
			_ = _out1
			_out1 = (((_this).Chains()).Get(_4_chainID).(*m_ChainContainer.ChainContainer)).OptimisticAt(ts)
			_5_chainResult = _out1
			if (_5_chainResult).Is_None() {
				result = m_Types.Companion_Option_.Create_None_()
				return result
			}
			_0_blocks = (_0_blocks).Update(_4_chainID, ((_5_chainResult).Dtor_value().(m_ChainContainer.OptimisticAtResult)).Dtor_l2Block())
			_1_l1Heads = (_1_l1Heads).Update(_4_chainID, ((_5_chainResult).Dtor_value().(m_ChainContainer.OptimisticAtResult)).Dtor_l1Head())
			if ((((_this).LogsDBs()).Get(_4_chainID).(*m_LogsDB.LogsDB)).LatestSealedBlock()).Is_Some() {
				var _6_latestBlock m_Types.BlockID
				_ = _6_latestBlock
				_6_latestBlock = ((((_this).LogsDBs()).Get(_4_chainID).(*m_LogsDB.LogsDB)).LatestSealedBlock()).Dtor_value().(m_Types.BlockID)
				var _7_newBlock m_Types.BlockID
				_ = _7_newBlock
				_7_newBlock = ((_5_chainResult).Dtor_value().(m_ChainContainer.OptimisticAtResult)).Dtor_l2Block()
			}
		}
		result = m_Types.Companion_Option_.Create_Some_(m_Types.Companion_ChainsReadyResult_.Create_ChainsReadyResult_(_0_blocks, _1_l1Heads))
		return result
	}
}
func (_this *Interop) Verify(ts _dafny.Int, blocksAtTS _dafny.Map, l1Heads _dafny.Map) m_Types.Result {
	{
		var result m_Types.Result = m_Types.Companion_Result_.Default()
		_ = result
		var _0_view *FrontierView
		_ = _0_view
		var _out0 *FrontierView
		_ = _out0
		_out0 = (_this).ResolveFrontierVerificationView(blocksAtTS)
		_0_view = _out0
		var _out1 m_Types.Result
		_ = _out1
		_out1 = (_this).VerifyMessages(ts, blocksAtTS, l1Heads, _0_view)
		result = _out1
		var _1_cycleResult m_Types.Result
		_ = _1_cycleResult
		var _out2 m_Types.Result
		_ = _out2
		_out2 = (_this).VerifyCycles(ts, blocksAtTS, _0_view)
		_1_cycleResult = _out2
		var _2_dt__update__tmp_h0 m_Types.Result = result
		_ = _2_dt__update__tmp_h0
		var _3_dt__update_hinvalidHeads_h0 _dafny.Map = ((result).Dtor_invalidHeads()).Merge((_1_cycleResult).Dtor_invalidHeads())
		_ = _3_dt__update_hinvalidHeads_h0
		result = m_Types.Companion_Result_.Create_Result_((_2_dt__update__tmp_h0).Dtor_timestamp(), (_2_dt__update__tmp_h0).Dtor_l1Inclusion(), (_2_dt__update__tmp_h0).Dtor_l2Heads(), _3_dt__update_hinvalidHeads_h0)
		return result
	}
}
func (_this *Interop) ApplyPendingTransition(pending m_Types.PendingTransition) m_Types.Option {
	{
		var madeProgress m_Types.Option = m_Types.Companion_Option_.Default()
		_ = madeProgress
		if ((pending).Dtor_decision()).Equals(m_Types.Companion_Decision_.Create_Rewind_()) {
			(_this).CurrentL1 = m_Types.Companion_BlockID_.Create_BlockID_(_dafny.Zero, _dafny.Zero)
			var _0_rewindPlan m_Types.RewindPlan
			_ = _0_rewindPlan
			_0_rewindPlan = ((pending).Dtor_rewind()).Dtor_value().(m_Types.RewindPlan)
			var _1_rewindOk bool
			_ = _1_rewindOk
			var _out0 bool
			_ = _out0
			_out0 = (_this).ApplyRewindPlan(_0_rewindPlan)
			_1_rewindOk = _out0
			if !(_1_rewindOk) {
				madeProgress = m_Types.Companion_Option_.Create_None_()
				return madeProgress
			}
			((_this).VerifiedDB()).ClearPendingTransition()
			madeProgress = m_Types.Companion_Option_.Create_Some_(false)
		} else if ((pending).Dtor_decision()).Equals(m_Types.Companion_Decision_.Create_Invalidate_()) {
			var _2_failedAny bool
			_ = _2_failedAny
			_2_failedAny = false
			var _3_invSeq _dafny.Sequence
			_ = _3_invSeq
			var _out1 _dafny.Sequence
			_ = _out1
			_out1 = m_Types.Companion_Default___.Enumerate(((((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_invalidHeads()).Keys())
			_3_invSeq = _out1
			var _hi0 _dafny.Int = _dafny.IntOfUint32((_3_invSeq).Cardinality())
			_ = _hi0
			for _4_i := _dafny.Zero; _4_i.Cmp(_hi0) < 0; _4_i = _4_i.Plus(_dafny.One) {
				var _5_chainID _dafny.Int
				_ = _5_chainID
				_5_chainID = (_3_invSeq).Select((_4_i).Uint32()).(_dafny.Int)
				if ((_this).Chains()).Contains(_5_chainID) {
					var _6_ok bool
					_ = _6_ok
					var _out2 bool
					_ = _out2
					_out2 = (((_this).Chains()).Get(_5_chainID).(*m_ChainContainer.ChainContainer)).InvalidateBlock(((((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_invalidHeads()).Get(_5_chainID).(m_Types.BlockID), (((pending).Dtor_result()).Dtor_value().(m_Types.Result)).Dtor_timestamp())
					_6_ok = _out2
					if !(_6_ok) {
						_2_failedAny = true
					}
				}
			}
			if _2_failedAny {
				madeProgress = m_Types.Companion_Option_.Create_None_()
				return madeProgress
			}
			((_this).VerifiedDB()).ClearPendingTransition()
			madeProgress = m_Types.Companion_Option_.Create_Some_(false)
		} else {
			var _7_result m_Types.Result
			_ = _7_result
			_7_result = ((pending).Dtor_result()).Dtor_value().(m_Types.Result)
			var _8_persistOk bool
			_ = _8_persistOk
			var _out3 bool
			_ = _out3
			_out3 = (_this).PersistFrontierLogs((_7_result).Dtor_timestamp(), (_7_result).Dtor_l2Heads())
			_8_persistOk = _out3
			if !(_8_persistOk) {
				madeProgress = m_Types.Companion_Option_.Create_None_()
				return madeProgress
			}
			((_this).VerifiedDB()).Commit(m_Types.Companion_VerifiedResult_.Create_VerifiedResult_((_7_result).Dtor_timestamp(), (_7_result).Dtor_l1Inclusion(), (_7_result).Dtor_l2Heads()))
			((_this).VerifiedDB()).ClearPendingTransition()
			(_this).CurrentL1 = (_7_result).Dtor_l1Inclusion()
			madeProgress = m_Types.Companion_Option_.Create_Some_(true)
		}
		return madeProgress
	}
}
func (_this *Interop) ApplyRewindPlan(plan m_Types.RewindPlan) bool {
	{
		var success bool = false
		_ = success
		var _0___v2 bool
		_ = _0___v2
		var _out0 bool
		_ = _out0
		_out0 = ((_this).VerifiedDB()).Rewind((plan).Dtor_rewindAtOrAfter())
		_0___v2 = _out0
		var _1_chainIDs _dafny.Sequence
		_ = _1_chainIDs
		var _out1 _dafny.Sequence
		_ = _out1
		_out1 = m_Types.Companion_Default___.Enumerate(((_this).Chains()).Keys())
		_1_chainIDs = _out1
		var _2_enginesOk bool
		_ = _2_enginesOk
		var _out2 bool
		_ = _out2
		_out2 = (_this).RewindChainEngines(plan, _1_chainIDs)
		_2_enginesOk = _out2
		if !(_2_enginesOk) {
			success = false
			return success
		}
		if ((plan).Dtor_resetAllChainsTo()).Is_None() {
			(_this).ClearLogsDBs(plan, _1_chainIDs)
		} else {
			(_this).RewindLogsDBs(plan, _1_chainIDs)
		}
		success = true
		return success
	}
}
func (_this *Interop) RewindChainEngines(plan m_Types.RewindPlan, chainIDs _dafny.Sequence) bool {
	{
		var success bool = false
		_ = success
		var _0_failedAny bool
		_ = _0_failedAny
		_0_failedAny = false
		var _hi0 _dafny.Int = _dafny.IntOfUint32((chainIDs).Cardinality())
		_ = _hi0
		for _1_i := _dafny.Zero; _1_i.Cmp(_hi0) < 0; _1_i = _1_i.Plus(_dafny.One) {
			(((_this).Chains()).Get((chainIDs).Select((_1_i).Uint32()).(_dafny.Int)).(*m_ChainContainer.ChainContainer)).PruneDeniedAtOrAfterTimestamp((plan).Dtor_rewindAtOrAfter())
			if ((plan).Dtor_resetAllChainsTo()).Is_Some() {
				var _2_ok bool
				_ = _2_ok
				var _out0 bool
				_ = _out0
				_out0 = (((_this).Chains()).Get((chainIDs).Select((_1_i).Uint32()).(_dafny.Int)).(*m_ChainContainer.ChainContainer)).RewindEngine(((plan).Dtor_resetAllChainsTo()).Dtor_value().(_dafny.Int))
				_2_ok = _out0
				if !(_2_ok) {
					_0_failedAny = true
				}
			}
		}
		success = !(_0_failedAny)
		return success
	}
}
func (_this *Interop) ClearLogsDBs(plan m_Types.RewindPlan, chainIDs _dafny.Sequence) {
	{
		var _hi0 _dafny.Int = _dafny.IntOfUint32((chainIDs).Cardinality())
		_ = _hi0
		for _0_i := _dafny.Zero; _0_i.Cmp(_hi0) < 0; _0_i = _0_i.Plus(_dafny.One) {
			(((_this).LogsDBs()).Get((chainIDs).Select((_0_i).Uint32()).(_dafny.Int)).(*m_LogsDB.LogsDB)).Clear()
		}
	}
}
func (_this *Interop) RewindLogsDBs(plan m_Types.RewindPlan, chainIDs _dafny.Sequence) {
	{
		var _0_ts _dafny.Int
		_ = _0_ts
		_0_ts = ((plan).Dtor_resetAllChainsTo()).Dtor_value().(_dafny.Int)
		var _hi0 _dafny.Int = _dafny.IntOfUint32((chainIDs).Cardinality())
		_ = _hi0
		for _1_i := _dafny.Zero; _1_i.Cmp(_hi0) < 0; _1_i = _1_i.Plus(_dafny.One) {
			var _2_chainID _dafny.Int
			_ = _2_chainID
			_2_chainID = (chainIDs).Select((_1_i).Uint32()).(_dafny.Int)
			(((_this).LogsDBs()).Get(_2_chainID).(*m_LogsDB.LogsDB)).Rewind(((plan).Dtor_targetHeads()).Get(_2_chainID).(m_Types.BlockID))
		}
	}
}
func (_this *Interop) PersistFrontierLogs(ts _dafny.Int, blocksAtTS _dafny.Map) bool {
	{
		var success bool = false
		_ = success
		success = true
		var _0_chainIDs _dafny.Sequence
		_ = _0_chainIDs
		var _out0 _dafny.Sequence
		_ = _out0
		_out0 = m_Types.Companion_Default___.Enumerate((blocksAtTS).Keys())
		_0_chainIDs = _out0
		var _hi0 _dafny.Int = _dafny.IntOfUint32((_0_chainIDs).Cardinality())
		_ = _hi0
		for _1_i := _dafny.Zero; _1_i.Cmp(_hi0) < 0; _1_i = _1_i.Plus(_dafny.One) {
			var _2_chainID _dafny.Int
			_ = _2_chainID
			_2_chainID = (_0_chainIDs).Select((_1_i).Uint32()).(_dafny.Int)
			var _3_blockID m_Types.BlockID
			_ = _3_blockID
			_3_blockID = (blocksAtTS).Get(_2_chainID).(m_Types.BlockID)
			var _4_db *m_LogsDB.LogsDB
			_ = _4_db
			_4_db = ((_this).LogsDBs()).Get(_2_chainID).(*m_LogsDB.LogsDB)
			var _5_chain *m_ChainContainer.ChainContainer
			_ = _5_chain
			_5_chain = ((_this).Chains()).Get(_2_chainID).(*m_ChainContainer.ChainContainer)
			var _6_latestBlock m_Types.Option
			_ = _6_latestBlock
			_6_latestBlock = (_4_db).LatestSealedBlock()
			var _7_skip bool
			_ = _7_skip
			_7_skip = (_6_latestBlock).Equals(m_Types.Companion_Option_.Create_Some_(_3_blockID))
			if _7_skip {
			} else {
				var _8_fetchResult m_Types.Option
				_ = _8_fetchResult
				var _out1 m_Types.Option
				_ = _out1
				_out1 = (_5_chain).FetchReceipts(_3_blockID)
				_8_fetchResult = _out1
				if (_8_fetchResult).Is_None() {
					success = false
					return success
				}
				var _9_blockInfo m_Types.BlockInfo
				_ = _9_blockInfo
				_9_blockInfo = ((_8_fetchResult).Dtor_value().(m_ChainContainer.FetchReceiptsResult)).Dtor_info()
				var _10_logs m_Types.BlockLogs
				_ = _10_logs
				_10_logs = ((_8_fetchResult).Dtor_value().(m_ChainContainer.FetchReceiptsResult)).Dtor_logs()
				(_this).ProcessBlock(_2_chainID, _3_blockID, _9_blockInfo, _10_logs)
			}
		}
		return success
	}
}
func (_this *Interop) BuildPendingTransition(output m_Types.StepOutput, obs m_Types.RoundObservation) m_Types.PendingTransition {
	{
		var pendingTx m_Types.PendingTransition = m_Types.Companion_PendingTransition_.Default()
		_ = pendingTx
		if (output).Is_AdvanceOutput() {
			pendingTx = m_Types.Companion_PendingTransition_.Create_PendingTransition_(m_Types.Companion_Decision_.Create_Advance_(), m_Types.Companion_Option_.Create_Some_((output).Dtor_result()), m_Types.Companion_Option_.Create_None_())
		} else if (output).Is_InvalidateOutput() {
			pendingTx = m_Types.Companion_PendingTransition_.Create_PendingTransition_(m_Types.Companion_Decision_.Create_Invalidate_(), m_Types.Companion_Option_.Create_Some_((output).Dtor_result()), m_Types.Companion_Option_.Create_None_())
		} else {
			var _0_lastTS _dafny.Int
			_ = _0_lastTS
			_0_lastTS = ((obs).Dtor_lastVerifiedTS()).Dtor_value().(_dafny.Int)
			var _1_rewindPlan m_Types.RewindPlan = m_Types.Companion_RewindPlan_.Default()
			_ = _1_rewindPlan
			if (_0_lastTS).Cmp((_this).ActivationTimestamp()) <= 0 {
				_1_rewindPlan = m_Types.Companion_RewindPlan_.Create_RewindPlan_(_0_lastTS, m_Types.Companion_Option_.Create_None_(), _dafny.NewMapBuilder().ToMap())
			} else {
				var _2_prevResult m_Types.VerifiedResult
				_ = _2_prevResult
				_2_prevResult = ((_this).VerifiedDB()).Get((_0_lastTS).Minus(_dafny.One))
				_1_rewindPlan = m_Types.Companion_RewindPlan_.Create_RewindPlan_(_0_lastTS, m_Types.Companion_Option_.Create_Some_((_0_lastTS).Minus(_dafny.One)), (_2_prevResult).Dtor_l2Heads())
			}
			pendingTx = m_Types.Companion_PendingTransition_.Create_PendingTransition_(m_Types.Companion_Decision_.Create_Rewind_(), m_Types.Companion_Option_.Create_None_(), m_Types.Companion_Option_.Create_Some_(_1_rewindPlan))
		}
		return pendingTx
	}
}
func (_this *Interop) VerifyExecutingMessage(executingChain _dafny.Int, executingTimestamp _dafny.Int, execMsg m_Types.ExecutingMessage, view *FrontierView) bool {
	{
		var valid bool = false
		_ = valid
		if (!((_this).LogsDBs()).Contains((execMsg).Dtor_chainID())) || (!((_this).Chains()).Contains((execMsg).Dtor_chainID())) {
			valid = false
			return valid
		}
		if (executingTimestamp).Cmp(((_this).ActivationTimestamp()).Plus((((_this).Chains()).Get(executingChain).(*m_ChainContainer.ChainContainer)).BlockTime())) < 0 {
			valid = false
			return valid
		}
		if ((execMsg).Dtor_timestamp()).Cmp(((_this).ActivationTimestamp()).Plus((((_this).Chains()).Get((execMsg).Dtor_chainID()).(*m_ChainContainer.ChainContainer)).BlockTime())) < 0 {
			valid = false
			return valid
		}
		if ((execMsg).Dtor_timestamp()).Cmp(executingTimestamp) > 0 {
			valid = false
			return valid
		}
		if (((execMsg).Dtor_timestamp()).Plus((_this).MessageExpiryWindow())).Cmp(executingTimestamp) < 0 {
			valid = false
			return valid
		}
		var _0_query m_Types.ContainsQuery
		_ = _0_query
		_0_query = m_Types.Companion_ContainsQuery_.Create_ContainsQuery_((execMsg).Dtor_blockNum(), (execMsg).Dtor_logIdx(), (execMsg).Dtor_timestamp(), (execMsg).Dtor_checksum())
		if ((execMsg).Dtor_timestamp()).Cmp(executingTimestamp) == 0 {
			valid = (view).Contains((execMsg).Dtor_chainID(), _0_query)
			return valid
		}
		valid = (((_this).LogsDBs()).Get((execMsg).Dtor_chainID()).(*m_LogsDB.LogsDB)).Contains(_0_query)
		return valid
	}
}
func (_this *Interop) VerifyMessages(ts _dafny.Int, blocksAtTS _dafny.Map, l1Heads _dafny.Map, view *FrontierView) m_Types.Result {
	{
		var result m_Types.Result = m_Types.Companion_Result_.Default()
		_ = result
		var _0_l1Inclusion m_Types.BlockID
		_ = _0_l1Inclusion
		var _out0 m_Types.BlockID
		_ = _out0
		_out0 = (_this).ComputeL1Inclusion(blocksAtTS, l1Heads)
		_0_l1Inclusion = _out0
		result = m_Types.Companion_Result_.Create_Result_(ts, _0_l1Inclusion, blocksAtTS, _dafny.NewMapBuilder().ToMap())
		var _1_chainIDs _dafny.Sequence
		_ = _1_chainIDs
		var _out1 _dafny.Sequence
		_ = _out1
		_out1 = m_Types.Companion_Default___.Enumerate((blocksAtTS).Keys())
		_1_chainIDs = _out1
		var _hi0 _dafny.Int = _dafny.IntOfUint32((_1_chainIDs).Cardinality())
		_ = _hi0
		for _2_i := _dafny.Zero; _2_i.Cmp(_hi0) < 0; _2_i = _2_i.Plus(_dafny.One) {
			var _3_chainID _dafny.Int
			_ = _3_chainID
			_3_chainID = (_1_chainIDs).Select((_2_i).Uint32()).(_dafny.Int)
			var _4_expectedBlock m_Types.BlockID
			_ = _4_expectedBlock
			_4_expectedBlock = (blocksAtTS).Get(_3_chainID).(m_Types.BlockID)
			var _5_block m_Types.BlockInfo
			_ = _5_block
			_5_block = (view).BlockInfo(_3_chainID)
			var _6_logs m_Types.BlockLogs
			_ = _6_logs
			_6_logs = (view).BlockLogs(_3_chainID)
			var _7_blockValid bool
			_ = _7_blockValid
			_7_blockValid = true
			var _8_logIdxs _dafny.Sequence
			_ = _8_logIdxs
			var _out2 _dafny.Sequence
			_ = _out2
			_out2 = m_Types.Companion_Default___.Enumerate(((_6_logs).Dtor_execMsgs()).Keys())
			_8_logIdxs = _out2
			var _9_j _dafny.Int
			_ = _9_j
			_9_j = _dafny.Zero
			for ((_9_j).Cmp(_dafny.IntOfUint32((_8_logIdxs).Cardinality())) < 0) && (_7_blockValid) {
				var _10_logIdx _dafny.Int
				_ = _10_logIdx
				_10_logIdx = (_8_logIdxs).Select((_9_j).Uint32()).(_dafny.Int)
				var _11_execMsg m_Types.ExecutingMessage
				_ = _11_execMsg
				_11_execMsg = ((_6_logs).Dtor_execMsgs()).Get(_10_logIdx).(m_Types.ExecutingMessage)
				var _12_ok bool
				_ = _12_ok
				var _out3 bool
				_ = _out3
				_out3 = (_this).VerifyExecutingMessage(_3_chainID, (_5_block).Dtor_timestamp(), _11_execMsg, view)
				_12_ok = _out3
				if !(_12_ok) {
					_7_blockValid = false
				} else {
				}
				_9_j = (_9_j).Plus(_dafny.One)
			}
			if !(_7_blockValid) {
				var _13_dt__update__tmp_h0 m_Types.Result = result
				_ = _13_dt__update__tmp_h0
				var _14_dt__update_hinvalidHeads_h0 _dafny.Map = ((result).Dtor_invalidHeads()).Update(_3_chainID, _4_expectedBlock)
				_ = _14_dt__update_hinvalidHeads_h0
				result = m_Types.Companion_Result_.Create_Result_((_13_dt__update__tmp_h0).Dtor_timestamp(), (_13_dt__update__tmp_h0).Dtor_l1Inclusion(), (_13_dt__update__tmp_h0).Dtor_l2Heads(), _14_dt__update_hinvalidHeads_h0)
			} else {
			}
		}
		return result
	}
}
func (_this *Interop) VerifyCycles(ts _dafny.Int, blocksAtTS _dafny.Map, view *FrontierView) m_Types.Result {
	{
		var result m_Types.Result = m_Types.Companion_Result_.Default()
		_ = result
		return result
	}
}
func (_this *Interop) ResolveFrontierVerificationView(blocksAtTS _dafny.Map) *FrontierView {
	{
		var view *FrontierView = (*FrontierView)(nil)
		_ = view
		return view
	}
}
func (_this *Interop) ComputeL1Inclusion(blocksAtTS _dafny.Map, l1Heads _dafny.Map) m_Types.BlockID {
	{
		var l1Inclusion m_Types.BlockID = m_Types.Companion_BlockID_.Default()
		_ = l1Inclusion
		return l1Inclusion
	}
}
func (_this *Interop) ProcessBlock(chainID _dafny.Int, blockID m_Types.BlockID, info m_Types.BlockInfo, logs m_Types.BlockLogs) {
	{
	}
}
func (_this *Interop) CheckL1Consistent(l1Heads _dafny.Map, lastVerified m_Types.Option) (bool, bool) {
	{
		var l1Consistent bool = false
		_ = l1Consistent
		var l2sConsistent bool = false
		_ = l2sConsistent
		return l1Consistent, l2sConsistent
	}
}
func (_this *Interop) RefreshCurrentL1OnWait() {
	{
	}
}
func (_this *Interop) VerifiedDB() *m_VerifiedDB.VerifiedDB {
	{
		return _this._verifiedDB
	}
}
func (_this *Interop) LogsDBs() _dafny.Map {
	{
		return _this._logsDBs
	}
}
func (_this *Interop) ActivationTimestamp() _dafny.Int {
	{
		return _this._activationTimestamp
	}
}
func (_this *Interop) MessageExpiryWindow() _dafny.Int {
	{
		return _this._messageExpiryWindow
	}
}
func (_this *Interop) Chains() _dafny.Map {
	{
		return _this._chains
	}
}

// End of class Interop
