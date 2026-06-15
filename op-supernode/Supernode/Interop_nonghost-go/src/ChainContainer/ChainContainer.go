// Package ChainContainer
// Dafny module ChainContainer compiled into Go

package ChainContainer

import (
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

type Dummy__ struct{}

// Definition of datatype OptimisticAtResult
type OptimisticAtResult struct {
	Data_OptimisticAtResult_
}

func (_this OptimisticAtResult) Get_() Data_OptimisticAtResult_ {
	return _this.Data_OptimisticAtResult_
}

type Data_OptimisticAtResult_ interface {
	isOptimisticAtResult()
}

type CompanionStruct_OptimisticAtResult_ struct {
}

var Companion_OptimisticAtResult_ = CompanionStruct_OptimisticAtResult_{}

type OptimisticAtResult_OptimisticAtResult struct {
	L2Block m_Types.BlockID
	L1Head  m_Types.BlockID
}

func (OptimisticAtResult_OptimisticAtResult) isOptimisticAtResult() {}

func (CompanionStruct_OptimisticAtResult_) Create_OptimisticAtResult_(L2Block m_Types.BlockID, L1Head m_Types.BlockID) OptimisticAtResult {
	return OptimisticAtResult{OptimisticAtResult_OptimisticAtResult{L2Block, L1Head}}
}

func (_this OptimisticAtResult) Is_OptimisticAtResult() bool {
	_, ok := _this.Get_().(OptimisticAtResult_OptimisticAtResult)
	return ok
}

func (CompanionStruct_OptimisticAtResult_) Default() OptimisticAtResult {
	return Companion_OptimisticAtResult_.Create_OptimisticAtResult_(m_Types.Companion_BlockID_.Default(), m_Types.Companion_BlockID_.Default())
}

func (_this OptimisticAtResult) Dtor_l2Block() m_Types.BlockID {
	return _this.Get_().(OptimisticAtResult_OptimisticAtResult).L2Block
}

func (_this OptimisticAtResult) Dtor_l1Head() m_Types.BlockID {
	return _this.Get_().(OptimisticAtResult_OptimisticAtResult).L1Head
}

func (_this OptimisticAtResult) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case OptimisticAtResult_OptimisticAtResult:
		{
			return "ChainContainer.OptimisticAtResult.OptimisticAtResult" + "(" + _dafny.String(data.L2Block) + ", " + _dafny.String(data.L1Head) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this OptimisticAtResult) Equals(other OptimisticAtResult) bool {
	switch data1 := _this.Get_().(type) {
	case OptimisticAtResult_OptimisticAtResult:
		{
			data2, ok := other.Get_().(OptimisticAtResult_OptimisticAtResult)
			return ok && data1.L2Block.Equals(data2.L2Block) && data1.L1Head.Equals(data2.L1Head)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this OptimisticAtResult) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(OptimisticAtResult)
	return ok && _this.Equals(typed)
}

func Type_OptimisticAtResult_() _dafny.TypeDescriptor {
	return type_OptimisticAtResult_{}
}

type type_OptimisticAtResult_ struct {
}

func (_this type_OptimisticAtResult_) Default() interface{} {
	return Companion_OptimisticAtResult_.Default()
}

func (_this type_OptimisticAtResult_) String() string {
	return "ChainContainer.OptimisticAtResult"
}
func (_this OptimisticAtResult) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = OptimisticAtResult{}

// End of datatype OptimisticAtResult

// Definition of datatype FetchReceiptsResult
type FetchReceiptsResult struct {
	Data_FetchReceiptsResult_
}

func (_this FetchReceiptsResult) Get_() Data_FetchReceiptsResult_ {
	return _this.Data_FetchReceiptsResult_
}

type Data_FetchReceiptsResult_ interface {
	isFetchReceiptsResult()
}

type CompanionStruct_FetchReceiptsResult_ struct {
}

var Companion_FetchReceiptsResult_ = CompanionStruct_FetchReceiptsResult_{}

type FetchReceiptsResult_FetchReceiptsResult struct {
	Info m_Types.BlockInfo
	Logs m_Types.BlockLogs
}

func (FetchReceiptsResult_FetchReceiptsResult) isFetchReceiptsResult() {}

func (CompanionStruct_FetchReceiptsResult_) Create_FetchReceiptsResult_(Info m_Types.BlockInfo, Logs m_Types.BlockLogs) FetchReceiptsResult {
	return FetchReceiptsResult{FetchReceiptsResult_FetchReceiptsResult{Info, Logs}}
}

func (_this FetchReceiptsResult) Is_FetchReceiptsResult() bool {
	_, ok := _this.Get_().(FetchReceiptsResult_FetchReceiptsResult)
	return ok
}

func (CompanionStruct_FetchReceiptsResult_) Default() FetchReceiptsResult {
	return Companion_FetchReceiptsResult_.Create_FetchReceiptsResult_(m_Types.Companion_BlockInfo_.Default(), m_Types.Companion_BlockLogs_.Default())
}

func (_this FetchReceiptsResult) Dtor_info() m_Types.BlockInfo {
	return _this.Get_().(FetchReceiptsResult_FetchReceiptsResult).Info
}

func (_this FetchReceiptsResult) Dtor_logs() m_Types.BlockLogs {
	return _this.Get_().(FetchReceiptsResult_FetchReceiptsResult).Logs
}

func (_this FetchReceiptsResult) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case FetchReceiptsResult_FetchReceiptsResult:
		{
			return "ChainContainer.FetchReceiptsResult.FetchReceiptsResult" + "(" + _dafny.String(data.Info) + ", " + _dafny.String(data.Logs) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this FetchReceiptsResult) Equals(other FetchReceiptsResult) bool {
	switch data1 := _this.Get_().(type) {
	case FetchReceiptsResult_FetchReceiptsResult:
		{
			data2, ok := other.Get_().(FetchReceiptsResult_FetchReceiptsResult)
			return ok && data1.Info.Equals(data2.Info) && data1.Logs.Equals(data2.Logs)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this FetchReceiptsResult) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(FetchReceiptsResult)
	return ok && _this.Equals(typed)
}

func Type_FetchReceiptsResult_() _dafny.TypeDescriptor {
	return type_FetchReceiptsResult_{}
}

type type_FetchReceiptsResult_ struct {
}

func (_this type_FetchReceiptsResult_) Default() interface{} {
	return Companion_FetchReceiptsResult_.Default()
}

func (_this type_FetchReceiptsResult_) String() string {
	return "ChainContainer.FetchReceiptsResult"
}
func (_this FetchReceiptsResult) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = FetchReceiptsResult{}

// End of datatype FetchReceiptsResult

// Definition of class ChainContainer
type ChainContainer struct {
	dummy byte
}

func New_ChainContainer_() *ChainContainer {
	_this := ChainContainer{}

	return &_this
}

type CompanionStruct_ChainContainer_ struct {
}

var Companion_ChainContainer_ = CompanionStruct_ChainContainer_{}

func (_this *ChainContainer) Equals(other *ChainContainer) bool {
	return _this == other
}

func (_this *ChainContainer) EqualsGeneric(x interface{}) bool {
	other, ok := x.(*ChainContainer)
	return ok && _this.Equals(other)
}

func (*ChainContainer) String() string {
	return "ChainContainer.ChainContainer"
}

func Type_ChainContainer_() _dafny.TypeDescriptor {
	return type_ChainContainer_{}
}

type type_ChainContainer_ struct {
}

func (_this type_ChainContainer_) Default() interface{} {
	return (*ChainContainer)(nil)
}

func (_this type_ChainContainer_) String() string {
	return "ChainContainer.ChainContainer"
}
func (_this *ChainContainer) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = &ChainContainer{}

func (_this *ChainContainer) OptimisticAt(ts _dafny.Int) m_Types.Option {
	{
		var result m_Types.Option = m_Types.Companion_Option_.Default()
		_ = result
		return result
	}
}
func (_this *ChainContainer) PruneDeniedAtOrAfterTimestamp(timestamp _dafny.Int) {
	{
	}
}
func (_this *ChainContainer) RewindEngine(resetTo _dafny.Int) bool {
	{
		var success bool = false
		_ = success
		return success
	}
}
func (_this *ChainContainer) InvalidateBlock(blockID m_Types.BlockID, timestamp _dafny.Int) bool {
	{
		var success bool = false
		_ = success
		return success
	}
}
func (_this *ChainContainer) FetchReceipts(blockID m_Types.BlockID) m_Types.Option {
	{
		var result m_Types.Option = m_Types.Companion_Option_.Default()
		_ = result
		return result
	}
}
func (_this *ChainContainer) BlockTime() _dafny.Int {
	{
		return _dafny.Zero
	}
}
func (_this *ChainContainer) BlockInfo(blockID m_Types.BlockID) m_Types.Option {
	{
		return m_Types.Companion_Option_.Create_None_()
	}
}
func (_this *ChainContainer) BlockLogs(blockID m_Types.BlockID) m_Types.Option {
	{
		return m_Types.Companion_Option_.Create_None_()
	}
}

// End of class ChainContainer
