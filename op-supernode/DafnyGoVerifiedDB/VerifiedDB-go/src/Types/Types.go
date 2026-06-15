// Package Types
// Dafny module Types compiled into Go

package Types

import (
	m__System "System_"
	_dafny "dafny"
	os "os"
)

var _ = os.Args
var _ _dafny.Dummy__
var _ m__System.Dummy__

type Dummy__ struct{}

// Definition of class Default__
type Default__ struct {
	dummy byte
}

func New_Default___() *Default__ {
	_this := Default__{}

	return &_this
}

type CompanionStruct_Default___ struct {
}

var Companion_Default___ = CompanionStruct_Default___{}

func (_this *Default__) Equals(other *Default__) bool {
	return _this == other
}

func (_this *Default__) EqualsGeneric(x interface{}) bool {
	other, ok := x.(*Default__)
	return ok && _this.Equals(other)
}

func (*Default__) String() string {
	return "Types.Default__"
}
func (_this *Default__) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = &Default__{}

func (_static *CompanionStruct_Default___) Enumerate(s _dafny.Set) _dafny.Sequence {
	var result _dafny.Sequence = _dafny.EmptySeq
	_ = result
	result = _dafny.SeqOf()
	var _0_remaining _dafny.Set
	_ = _0_remaining
	_0_remaining = s
	for !(_0_remaining).Equals(_dafny.SetOf()) {
		var _1_x _dafny.Int
		_ = _1_x
		{
			for _iter0 := _dafny.Iterate((_0_remaining).Elements()); ; {
				_assign_such_that_0, _ok0 := _iter0()
				if !_ok0 {
					break
				}
				_1_x = interface{}(_assign_such_that_0).(_dafny.Int)
				if m__System.Companion_Nat_.Is_(_1_x) {
					if (_0_remaining).Contains(_1_x) {
						goto L_ASSIGN_SUCH_THAT_0
					}
				}
			}
			panic("assign-such-that search produced no value")
			goto L_ASSIGN_SUCH_THAT_0
		}
	L_ASSIGN_SUCH_THAT_0:
		result = _dafny.Companion_Sequence_.Concatenate(result, _dafny.SeqOf(_1_x))
		_0_remaining = (_0_remaining).Difference(_dafny.SetOf(_1_x))
	}
	return result
}
func (_static *CompanionStruct_Default___) CHAIN__IDS() _dafny.Set {
	return _dafny.EmptySet
}
func (_static *CompanionStruct_Default___) ACTIVATION__TIMESTAMP() _dafny.Int {
	return _dafny.Zero
}
func (_static *CompanionStruct_Default___) MESSAGE__EXPIRY__WINDOW() _dafny.Int {
	return _dafny.Zero
}

// End of class Default__

// Definition of datatype BlockID
type BlockID struct {
	Data_BlockID_
}

func (_this BlockID) Get_() Data_BlockID_ {
	return _this.Data_BlockID_
}

type Data_BlockID_ interface {
	isBlockID()
}

type CompanionStruct_BlockID_ struct {
}

var Companion_BlockID_ = CompanionStruct_BlockID_{}

type BlockID_BlockID struct {
	Number _dafny.Int
	Hash   _dafny.Int
}

func (BlockID_BlockID) isBlockID() {}

func (CompanionStruct_BlockID_) Create_BlockID_(Number _dafny.Int, Hash _dafny.Int) BlockID {
	return BlockID{BlockID_BlockID{Number, Hash}}
}

func (_this BlockID) Is_BlockID() bool {
	_, ok := _this.Get_().(BlockID_BlockID)
	return ok
}

func (CompanionStruct_BlockID_) Default() BlockID {
	return Companion_BlockID_.Create_BlockID_(_dafny.Zero, _dafny.Zero)
}

func (_this BlockID) Dtor_number() _dafny.Int {
	return _this.Get_().(BlockID_BlockID).Number
}

func (_this BlockID) Dtor_hash() _dafny.Int {
	return _this.Get_().(BlockID_BlockID).Hash
}

func (_this BlockID) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case BlockID_BlockID:
		{
			return "Types.BlockID.BlockID" + "(" + _dafny.String(data.Number) + ", " + _dafny.String(data.Hash) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this BlockID) Equals(other BlockID) bool {
	switch data1 := _this.Get_().(type) {
	case BlockID_BlockID:
		{
			data2, ok := other.Get_().(BlockID_BlockID)
			return ok && data1.Number.Cmp(data2.Number) == 0 && data1.Hash.Cmp(data2.Hash) == 0
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this BlockID) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(BlockID)
	return ok && _this.Equals(typed)
}

func Type_BlockID_() _dafny.TypeDescriptor {
	return type_BlockID_{}
}

type type_BlockID_ struct {
}

func (_this type_BlockID_) Default() interface{} {
	return Companion_BlockID_.Default()
}

func (_this type_BlockID_) String() string {
	return "Types.BlockID"
}
func (_this BlockID) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = BlockID{}

// End of datatype BlockID

// Definition of datatype BlockInfo
type BlockInfo struct {
	Data_BlockInfo_
}

func (_this BlockInfo) Get_() Data_BlockInfo_ {
	return _this.Data_BlockInfo_
}

type Data_BlockInfo_ interface {
	isBlockInfo()
}

type CompanionStruct_BlockInfo_ struct {
}

var Companion_BlockInfo_ = CompanionStruct_BlockInfo_{}

type BlockInfo_BlockInfo struct {
	Id         BlockID
	ParentHash _dafny.Int
	Timestamp  _dafny.Int
}

func (BlockInfo_BlockInfo) isBlockInfo() {}

func (CompanionStruct_BlockInfo_) Create_BlockInfo_(Id BlockID, ParentHash _dafny.Int, Timestamp _dafny.Int) BlockInfo {
	return BlockInfo{BlockInfo_BlockInfo{Id, ParentHash, Timestamp}}
}

func (_this BlockInfo) Is_BlockInfo() bool {
	_, ok := _this.Get_().(BlockInfo_BlockInfo)
	return ok
}

func (CompanionStruct_BlockInfo_) Default() BlockInfo {
	return Companion_BlockInfo_.Create_BlockInfo_(Companion_BlockID_.Default(), _dafny.Zero, _dafny.Zero)
}

func (_this BlockInfo) Dtor_id() BlockID {
	return _this.Get_().(BlockInfo_BlockInfo).Id
}

func (_this BlockInfo) Dtor_parentHash() _dafny.Int {
	return _this.Get_().(BlockInfo_BlockInfo).ParentHash
}

func (_this BlockInfo) Dtor_timestamp() _dafny.Int {
	return _this.Get_().(BlockInfo_BlockInfo).Timestamp
}

func (_this BlockInfo) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case BlockInfo_BlockInfo:
		{
			return "Types.BlockInfo.BlockInfo" + "(" + _dafny.String(data.Id) + ", " + _dafny.String(data.ParentHash) + ", " + _dafny.String(data.Timestamp) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this BlockInfo) Equals(other BlockInfo) bool {
	switch data1 := _this.Get_().(type) {
	case BlockInfo_BlockInfo:
		{
			data2, ok := other.Get_().(BlockInfo_BlockInfo)
			return ok && data1.Id.Equals(data2.Id) && data1.ParentHash.Cmp(data2.ParentHash) == 0 && data1.Timestamp.Cmp(data2.Timestamp) == 0
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this BlockInfo) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(BlockInfo)
	return ok && _this.Equals(typed)
}

func Type_BlockInfo_() _dafny.TypeDescriptor {
	return type_BlockInfo_{}
}

type type_BlockInfo_ struct {
}

func (_this type_BlockInfo_) Default() interface{} {
	return Companion_BlockInfo_.Default()
}

func (_this type_BlockInfo_) String() string {
	return "Types.BlockInfo"
}
func (_this BlockInfo) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = BlockInfo{}

// End of datatype BlockInfo

// Definition of datatype VerifiedResult
type VerifiedResult struct {
	Data_VerifiedResult_
}

func (_this VerifiedResult) Get_() Data_VerifiedResult_ {
	return _this.Data_VerifiedResult_
}

type Data_VerifiedResult_ interface {
	isVerifiedResult()
}

type CompanionStruct_VerifiedResult_ struct {
}

var Companion_VerifiedResult_ = CompanionStruct_VerifiedResult_{}

type VerifiedResult_VerifiedResult struct {
	Timestamp   _dafny.Int
	L1Inclusion BlockID
	L2Heads     _dafny.Map
}

func (VerifiedResult_VerifiedResult) isVerifiedResult() {}

func (CompanionStruct_VerifiedResult_) Create_VerifiedResult_(Timestamp _dafny.Int, L1Inclusion BlockID, L2Heads _dafny.Map) VerifiedResult {
	return VerifiedResult{VerifiedResult_VerifiedResult{Timestamp, L1Inclusion, L2Heads}}
}

func (_this VerifiedResult) Is_VerifiedResult() bool {
	_, ok := _this.Get_().(VerifiedResult_VerifiedResult)
	return ok
}

func (CompanionStruct_VerifiedResult_) Default() VerifiedResult {
	return Companion_VerifiedResult_.Create_VerifiedResult_(_dafny.Zero, Companion_BlockID_.Default(), _dafny.EmptyMap)
}

func (_this VerifiedResult) Dtor_timestamp() _dafny.Int {
	return _this.Get_().(VerifiedResult_VerifiedResult).Timestamp
}

func (_this VerifiedResult) Dtor_l1Inclusion() BlockID {
	return _this.Get_().(VerifiedResult_VerifiedResult).L1Inclusion
}

func (_this VerifiedResult) Dtor_l2Heads() _dafny.Map {
	return _this.Get_().(VerifiedResult_VerifiedResult).L2Heads
}

func (_this VerifiedResult) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case VerifiedResult_VerifiedResult:
		{
			return "Types.VerifiedResult.VerifiedResult" + "(" + _dafny.String(data.Timestamp) + ", " + _dafny.String(data.L1Inclusion) + ", " + _dafny.String(data.L2Heads) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this VerifiedResult) Equals(other VerifiedResult) bool {
	switch data1 := _this.Get_().(type) {
	case VerifiedResult_VerifiedResult:
		{
			data2, ok := other.Get_().(VerifiedResult_VerifiedResult)
			return ok && data1.Timestamp.Cmp(data2.Timestamp) == 0 && data1.L1Inclusion.Equals(data2.L1Inclusion) && data1.L2Heads.Equals(data2.L2Heads)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this VerifiedResult) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(VerifiedResult)
	return ok && _this.Equals(typed)
}

func Type_VerifiedResult_() _dafny.TypeDescriptor {
	return type_VerifiedResult_{}
}

type type_VerifiedResult_ struct {
}

func (_this type_VerifiedResult_) Default() interface{} {
	return Companion_VerifiedResult_.Default()
}

func (_this type_VerifiedResult_) String() string {
	return "Types.VerifiedResult"
}
func (_this VerifiedResult) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = VerifiedResult{}

// End of datatype VerifiedResult

// Definition of datatype Decision
type Decision struct {
	Data_Decision_
}

func (_this Decision) Get_() Data_Decision_ {
	return _this.Data_Decision_
}

type Data_Decision_ interface {
	isDecision()
}

type CompanionStruct_Decision_ struct {
}

var Companion_Decision_ = CompanionStruct_Decision_{}

type Decision_Wait struct {
}

func (Decision_Wait) isDecision() {}

func (CompanionStruct_Decision_) Create_Wait_() Decision {
	return Decision{Decision_Wait{}}
}

func (_this Decision) Is_Wait() bool {
	_, ok := _this.Get_().(Decision_Wait)
	return ok
}

type Decision_Advance struct {
}

func (Decision_Advance) isDecision() {}

func (CompanionStruct_Decision_) Create_Advance_() Decision {
	return Decision{Decision_Advance{}}
}

func (_this Decision) Is_Advance() bool {
	_, ok := _this.Get_().(Decision_Advance)
	return ok
}

type Decision_Invalidate struct {
}

func (Decision_Invalidate) isDecision() {}

func (CompanionStruct_Decision_) Create_Invalidate_() Decision {
	return Decision{Decision_Invalidate{}}
}

func (_this Decision) Is_Invalidate() bool {
	_, ok := _this.Get_().(Decision_Invalidate)
	return ok
}

type Decision_Rewind struct {
}

func (Decision_Rewind) isDecision() {}

func (CompanionStruct_Decision_) Create_Rewind_() Decision {
	return Decision{Decision_Rewind{}}
}

func (_this Decision) Is_Rewind() bool {
	_, ok := _this.Get_().(Decision_Rewind)
	return ok
}

func (CompanionStruct_Decision_) Default() Decision {
	return Companion_Decision_.Create_Wait_()
}

func (_ CompanionStruct_Decision_) AllSingletonConstructors() _dafny.Iterator {
	i := -1
	return func() (interface{}, bool) {
		i++
		switch i {
		case 0:
			return Companion_Decision_.Create_Wait_(), true
		case 1:
			return Companion_Decision_.Create_Advance_(), true
		case 2:
			return Companion_Decision_.Create_Invalidate_(), true
		case 3:
			return Companion_Decision_.Create_Rewind_(), true
		default:
			return Decision{}, false
		}
	}
}

func (_this Decision) String() string {
	switch _this.Get_().(type) {
	case nil:
		return "null"
	case Decision_Wait:
		{
			return "Types.Decision.Wait"
		}
	case Decision_Advance:
		{
			return "Types.Decision.Advance"
		}
	case Decision_Invalidate:
		{
			return "Types.Decision.Invalidate"
		}
	case Decision_Rewind:
		{
			return "Types.Decision.Rewind"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this Decision) Equals(other Decision) bool {
	switch _this.Get_().(type) {
	case Decision_Wait:
		{
			_, ok := other.Get_().(Decision_Wait)
			return ok
		}
	case Decision_Advance:
		{
			_, ok := other.Get_().(Decision_Advance)
			return ok
		}
	case Decision_Invalidate:
		{
			_, ok := other.Get_().(Decision_Invalidate)
			return ok
		}
	case Decision_Rewind:
		{
			_, ok := other.Get_().(Decision_Rewind)
			return ok
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this Decision) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(Decision)
	return ok && _this.Equals(typed)
}

func Type_Decision_() _dafny.TypeDescriptor {
	return type_Decision_{}
}

type type_Decision_ struct {
}

func (_this type_Decision_) Default() interface{} {
	return Companion_Decision_.Default()
}

func (_this type_Decision_) String() string {
	return "Types.Decision"
}
func (_this Decision) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = Decision{}

// End of datatype Decision

// Definition of datatype Option
type Option struct {
	Data_Option_
}

func (_this Option) Get_() Data_Option_ {
	return _this.Data_Option_
}

type Data_Option_ interface {
	isOption()
}

type CompanionStruct_Option_ struct {
}

var Companion_Option_ = CompanionStruct_Option_{}

type Option_None struct {
}

func (Option_None) isOption() {}

func (CompanionStruct_Option_) Create_None_() Option {
	return Option{Option_None{}}
}

func (_this Option) Is_None() bool {
	_, ok := _this.Get_().(Option_None)
	return ok
}

type Option_Some struct {
	Value interface{}
}

func (Option_Some) isOption() {}

func (CompanionStruct_Option_) Create_Some_(Value interface{}) Option {
	return Option{Option_Some{Value}}
}

func (_this Option) Is_Some() bool {
	_, ok := _this.Get_().(Option_Some)
	return ok
}

func (CompanionStruct_Option_) Default() Option {
	return Companion_Option_.Create_None_()
}

func (_this Option) Dtor_value() interface{} {
	return _this.Get_().(Option_Some).Value
}

func (_this Option) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case Option_None:
		{
			return "Types.Option.None"
		}
	case Option_Some:
		{
			return "Types.Option.Some" + "(" + _dafny.String(data.Value) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this Option) Equals(other Option) bool {
	switch data1 := _this.Get_().(type) {
	case Option_None:
		{
			_, ok := other.Get_().(Option_None)
			return ok
		}
	case Option_Some:
		{
			data2, ok := other.Get_().(Option_Some)
			return ok && _dafny.AreEqual(data1.Value, data2.Value)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this Option) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(Option)
	return ok && _this.Equals(typed)
}

func Type_Option_() _dafny.TypeDescriptor {
	return type_Option_{}
}

type type_Option_ struct {
}

func (_this type_Option_) Default() interface{} {
	return Companion_Option_.Default()
}

func (_this type_Option_) String() string {
	return "Types.Option"
}
func (_this Option) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = Option{}

// End of datatype Option

// Definition of datatype Result
type Result struct {
	Data_Result_
}

func (_this Result) Get_() Data_Result_ {
	return _this.Data_Result_
}

type Data_Result_ interface {
	isResult()
}

type CompanionStruct_Result_ struct {
}

var Companion_Result_ = CompanionStruct_Result_{}

type Result_Result struct {
	Timestamp    _dafny.Int
	L1Inclusion  BlockID
	L2Heads      _dafny.Map
	InvalidHeads _dafny.Map
}

func (Result_Result) isResult() {}

func (CompanionStruct_Result_) Create_Result_(Timestamp _dafny.Int, L1Inclusion BlockID, L2Heads _dafny.Map, InvalidHeads _dafny.Map) Result {
	return Result{Result_Result{Timestamp, L1Inclusion, L2Heads, InvalidHeads}}
}

func (_this Result) Is_Result() bool {
	_, ok := _this.Get_().(Result_Result)
	return ok
}

func (CompanionStruct_Result_) Default() Result {
	return Companion_Result_.Create_Result_(_dafny.Zero, Companion_BlockID_.Default(), _dafny.EmptyMap, _dafny.EmptyMap)
}

func (_this Result) Dtor_timestamp() _dafny.Int {
	return _this.Get_().(Result_Result).Timestamp
}

func (_this Result) Dtor_l1Inclusion() BlockID {
	return _this.Get_().(Result_Result).L1Inclusion
}

func (_this Result) Dtor_l2Heads() _dafny.Map {
	return _this.Get_().(Result_Result).L2Heads
}

func (_this Result) Dtor_invalidHeads() _dafny.Map {
	return _this.Get_().(Result_Result).InvalidHeads
}

func (_this Result) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case Result_Result:
		{
			return "Types.Result.Result" + "(" + _dafny.String(data.Timestamp) + ", " + _dafny.String(data.L1Inclusion) + ", " + _dafny.String(data.L2Heads) + ", " + _dafny.String(data.InvalidHeads) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this Result) Equals(other Result) bool {
	switch data1 := _this.Get_().(type) {
	case Result_Result:
		{
			data2, ok := other.Get_().(Result_Result)
			return ok && data1.Timestamp.Cmp(data2.Timestamp) == 0 && data1.L1Inclusion.Equals(data2.L1Inclusion) && data1.L2Heads.Equals(data2.L2Heads) && data1.InvalidHeads.Equals(data2.InvalidHeads)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this Result) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(Result)
	return ok && _this.Equals(typed)
}

func Type_Result_() _dafny.TypeDescriptor {
	return type_Result_{}
}

type type_Result_ struct {
}

func (_this type_Result_) Default() interface{} {
	return Companion_Result_.Default()
}

func (_this type_Result_) String() string {
	return "Types.Result"
}
func (_this Result) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = Result{}

// End of datatype Result

// Definition of datatype RewindPlan
type RewindPlan struct {
	Data_RewindPlan_
}

func (_this RewindPlan) Get_() Data_RewindPlan_ {
	return _this.Data_RewindPlan_
}

type Data_RewindPlan_ interface {
	isRewindPlan()
}

type CompanionStruct_RewindPlan_ struct {
}

var Companion_RewindPlan_ = CompanionStruct_RewindPlan_{}

type RewindPlan_RewindPlan struct {
	RewindAtOrAfter  _dafny.Int
	ResetAllChainsTo Option
	TargetHeads      _dafny.Map
}

func (RewindPlan_RewindPlan) isRewindPlan() {}

func (CompanionStruct_RewindPlan_) Create_RewindPlan_(RewindAtOrAfter _dafny.Int, ResetAllChainsTo Option, TargetHeads _dafny.Map) RewindPlan {
	return RewindPlan{RewindPlan_RewindPlan{RewindAtOrAfter, ResetAllChainsTo, TargetHeads}}
}

func (_this RewindPlan) Is_RewindPlan() bool {
	_, ok := _this.Get_().(RewindPlan_RewindPlan)
	return ok
}

func (CompanionStruct_RewindPlan_) Default() RewindPlan {
	return Companion_RewindPlan_.Create_RewindPlan_(_dafny.Zero, Companion_Option_.Default(), _dafny.EmptyMap)
}

func (_this RewindPlan) Dtor_rewindAtOrAfter() _dafny.Int {
	return _this.Get_().(RewindPlan_RewindPlan).RewindAtOrAfter
}

func (_this RewindPlan) Dtor_resetAllChainsTo() Option {
	return _this.Get_().(RewindPlan_RewindPlan).ResetAllChainsTo
}

func (_this RewindPlan) Dtor_targetHeads() _dafny.Map {
	return _this.Get_().(RewindPlan_RewindPlan).TargetHeads
}

func (_this RewindPlan) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case RewindPlan_RewindPlan:
		{
			return "Types.RewindPlan.RewindPlan" + "(" + _dafny.String(data.RewindAtOrAfter) + ", " + _dafny.String(data.ResetAllChainsTo) + ", " + _dafny.String(data.TargetHeads) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this RewindPlan) Equals(other RewindPlan) bool {
	switch data1 := _this.Get_().(type) {
	case RewindPlan_RewindPlan:
		{
			data2, ok := other.Get_().(RewindPlan_RewindPlan)
			return ok && data1.RewindAtOrAfter.Cmp(data2.RewindAtOrAfter) == 0 && data1.ResetAllChainsTo.Equals(data2.ResetAllChainsTo) && data1.TargetHeads.Equals(data2.TargetHeads)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this RewindPlan) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(RewindPlan)
	return ok && _this.Equals(typed)
}

func Type_RewindPlan_() _dafny.TypeDescriptor {
	return type_RewindPlan_{}
}

type type_RewindPlan_ struct {
}

func (_this type_RewindPlan_) Default() interface{} {
	return Companion_RewindPlan_.Default()
}

func (_this type_RewindPlan_) String() string {
	return "Types.RewindPlan"
}
func (_this RewindPlan) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = RewindPlan{}

// End of datatype RewindPlan

// Definition of datatype PendingTransition
type PendingTransition struct {
	Data_PendingTransition_
}

func (_this PendingTransition) Get_() Data_PendingTransition_ {
	return _this.Data_PendingTransition_
}

type Data_PendingTransition_ interface {
	isPendingTransition()
}

type CompanionStruct_PendingTransition_ struct {
}

var Companion_PendingTransition_ = CompanionStruct_PendingTransition_{}

type PendingTransition_PendingTransition struct {
	Decision Decision
	Result   Option
	Rewind   Option
}

func (PendingTransition_PendingTransition) isPendingTransition() {}

func (CompanionStruct_PendingTransition_) Create_PendingTransition_(Decision Decision, Result Option, Rewind Option) PendingTransition {
	return PendingTransition{PendingTransition_PendingTransition{Decision, Result, Rewind}}
}

func (_this PendingTransition) Is_PendingTransition() bool {
	_, ok := _this.Get_().(PendingTransition_PendingTransition)
	return ok
}

func (CompanionStruct_PendingTransition_) Default() PendingTransition {
	return Companion_PendingTransition_.Create_PendingTransition_(Companion_Decision_.Default(), Companion_Option_.Default(), Companion_Option_.Default())
}

func (_this PendingTransition) Dtor_decision() Decision {
	return _this.Get_().(PendingTransition_PendingTransition).Decision
}

func (_this PendingTransition) Dtor_result() Option {
	return _this.Get_().(PendingTransition_PendingTransition).Result
}

func (_this PendingTransition) Dtor_rewind() Option {
	return _this.Get_().(PendingTransition_PendingTransition).Rewind
}

func (_this PendingTransition) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case PendingTransition_PendingTransition:
		{
			return "Types.PendingTransition.PendingTransition" + "(" + _dafny.String(data.Decision) + ", " + _dafny.String(data.Result) + ", " + _dafny.String(data.Rewind) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this PendingTransition) Equals(other PendingTransition) bool {
	switch data1 := _this.Get_().(type) {
	case PendingTransition_PendingTransition:
		{
			data2, ok := other.Get_().(PendingTransition_PendingTransition)
			return ok && data1.Decision.Equals(data2.Decision) && data1.Result.Equals(data2.Result) && data1.Rewind.Equals(data2.Rewind)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this PendingTransition) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(PendingTransition)
	return ok && _this.Equals(typed)
}

func Type_PendingTransition_() _dafny.TypeDescriptor {
	return type_PendingTransition_{}
}

type type_PendingTransition_ struct {
}

func (_this type_PendingTransition_) Default() interface{} {
	return Companion_PendingTransition_.Default()
}

func (_this type_PendingTransition_) String() string {
	return "Types.PendingTransition"
}
func (_this PendingTransition) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = PendingTransition{}

// End of datatype PendingTransition

// Definition of datatype BlockSeal
type BlockSeal struct {
	Data_BlockSeal_
}

func (_this BlockSeal) Get_() Data_BlockSeal_ {
	return _this.Data_BlockSeal_
}

type Data_BlockSeal_ interface {
	isBlockSeal()
}

type CompanionStruct_BlockSeal_ struct {
}

var Companion_BlockSeal_ = CompanionStruct_BlockSeal_{}

type BlockSeal_BlockSeal struct {
	Id        BlockID
	Timestamp _dafny.Int
}

func (BlockSeal_BlockSeal) isBlockSeal() {}

func (CompanionStruct_BlockSeal_) Create_BlockSeal_(Id BlockID, Timestamp _dafny.Int) BlockSeal {
	return BlockSeal{BlockSeal_BlockSeal{Id, Timestamp}}
}

func (_this BlockSeal) Is_BlockSeal() bool {
	_, ok := _this.Get_().(BlockSeal_BlockSeal)
	return ok
}

func (CompanionStruct_BlockSeal_) Default() BlockSeal {
	return Companion_BlockSeal_.Create_BlockSeal_(Companion_BlockID_.Default(), _dafny.Zero)
}

func (_this BlockSeal) Dtor_id() BlockID {
	return _this.Get_().(BlockSeal_BlockSeal).Id
}

func (_this BlockSeal) Dtor_timestamp() _dafny.Int {
	return _this.Get_().(BlockSeal_BlockSeal).Timestamp
}

func (_this BlockSeal) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case BlockSeal_BlockSeal:
		{
			return "Types.BlockSeal.BlockSeal" + "(" + _dafny.String(data.Id) + ", " + _dafny.String(data.Timestamp) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this BlockSeal) Equals(other BlockSeal) bool {
	switch data1 := _this.Get_().(type) {
	case BlockSeal_BlockSeal:
		{
			data2, ok := other.Get_().(BlockSeal_BlockSeal)
			return ok && data1.Id.Equals(data2.Id) && data1.Timestamp.Cmp(data2.Timestamp) == 0
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this BlockSeal) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(BlockSeal)
	return ok && _this.Equals(typed)
}

func Type_BlockSeal_() _dafny.TypeDescriptor {
	return type_BlockSeal_{}
}

type type_BlockSeal_ struct {
}

func (_this type_BlockSeal_) Default() interface{} {
	return Companion_BlockSeal_.Default()
}

func (_this type_BlockSeal_) String() string {
	return "Types.BlockSeal"
}
func (_this BlockSeal) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = BlockSeal{}

// End of datatype BlockSeal

// Definition of datatype RoundObservation
type RoundObservation struct {
	Data_RoundObservation_
}

func (_this RoundObservation) Get_() Data_RoundObservation_ {
	return _this.Data_RoundObservation_
}

type Data_RoundObservation_ interface {
	isRoundObservation()
}

type CompanionStruct_RoundObservation_ struct {
}

var Companion_RoundObservation_ = CompanionStruct_RoundObservation_{}

type RoundObservation_RoundObservation struct {
	LastVerifiedTS Option
	LastVerified   Option
	NextTimestamp  _dafny.Int
	ChainsReady    bool
	BlocksAtTS     _dafny.Map
	L1Heads        _dafny.Map
	L1Consistent   bool
	L2sConsistent  bool
}

func (RoundObservation_RoundObservation) isRoundObservation() {}

func (CompanionStruct_RoundObservation_) Create_RoundObservation_(LastVerifiedTS Option, LastVerified Option, NextTimestamp _dafny.Int, ChainsReady bool, BlocksAtTS _dafny.Map, L1Heads _dafny.Map, L1Consistent bool, L2sConsistent bool) RoundObservation {
	return RoundObservation{RoundObservation_RoundObservation{LastVerifiedTS, LastVerified, NextTimestamp, ChainsReady, BlocksAtTS, L1Heads, L1Consistent, L2sConsistent}}
}

func (_this RoundObservation) Is_RoundObservation() bool {
	_, ok := _this.Get_().(RoundObservation_RoundObservation)
	return ok
}

func (CompanionStruct_RoundObservation_) Default() RoundObservation {
	return Companion_RoundObservation_.Create_RoundObservation_(Companion_Option_.Default(), Companion_Option_.Default(), _dafny.Zero, false, _dafny.EmptyMap, _dafny.EmptyMap, false, false)
}

func (_this RoundObservation) Dtor_lastVerifiedTS() Option {
	return _this.Get_().(RoundObservation_RoundObservation).LastVerifiedTS
}

func (_this RoundObservation) Dtor_lastVerified() Option {
	return _this.Get_().(RoundObservation_RoundObservation).LastVerified
}

func (_this RoundObservation) Dtor_nextTimestamp() _dafny.Int {
	return _this.Get_().(RoundObservation_RoundObservation).NextTimestamp
}

func (_this RoundObservation) Dtor_chainsReady() bool {
	return _this.Get_().(RoundObservation_RoundObservation).ChainsReady
}

func (_this RoundObservation) Dtor_blocksAtTS() _dafny.Map {
	return _this.Get_().(RoundObservation_RoundObservation).BlocksAtTS
}

func (_this RoundObservation) Dtor_l1Heads() _dafny.Map {
	return _this.Get_().(RoundObservation_RoundObservation).L1Heads
}

func (_this RoundObservation) Dtor_l1Consistent() bool {
	return _this.Get_().(RoundObservation_RoundObservation).L1Consistent
}

func (_this RoundObservation) Dtor_l2sConsistent() bool {
	return _this.Get_().(RoundObservation_RoundObservation).L2sConsistent
}

func (_this RoundObservation) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case RoundObservation_RoundObservation:
		{
			return "Types.RoundObservation.RoundObservation" + "(" + _dafny.String(data.LastVerifiedTS) + ", " + _dafny.String(data.LastVerified) + ", " + _dafny.String(data.NextTimestamp) + ", " + _dafny.String(data.ChainsReady) + ", " + _dafny.String(data.BlocksAtTS) + ", " + _dafny.String(data.L1Heads) + ", " + _dafny.String(data.L1Consistent) + ", " + _dafny.String(data.L2sConsistent) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this RoundObservation) Equals(other RoundObservation) bool {
	switch data1 := _this.Get_().(type) {
	case RoundObservation_RoundObservation:
		{
			data2, ok := other.Get_().(RoundObservation_RoundObservation)
			return ok && data1.LastVerifiedTS.Equals(data2.LastVerifiedTS) && data1.LastVerified.Equals(data2.LastVerified) && data1.NextTimestamp.Cmp(data2.NextTimestamp) == 0 && data1.ChainsReady == data2.ChainsReady && data1.BlocksAtTS.Equals(data2.BlocksAtTS) && data1.L1Heads.Equals(data2.L1Heads) && data1.L1Consistent == data2.L1Consistent && data1.L2sConsistent == data2.L2sConsistent
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this RoundObservation) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(RoundObservation)
	return ok && _this.Equals(typed)
}

func Type_RoundObservation_() _dafny.TypeDescriptor {
	return type_RoundObservation_{}
}

type type_RoundObservation_ struct {
}

func (_this type_RoundObservation_) Default() interface{} {
	return Companion_RoundObservation_.Default()
}

func (_this type_RoundObservation_) String() string {
	return "Types.RoundObservation"
}
func (_this RoundObservation) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = RoundObservation{}

// End of datatype RoundObservation

// Definition of datatype ChainsReadyResult
type ChainsReadyResult struct {
	Data_ChainsReadyResult_
}

func (_this ChainsReadyResult) Get_() Data_ChainsReadyResult_ {
	return _this.Data_ChainsReadyResult_
}

type Data_ChainsReadyResult_ interface {
	isChainsReadyResult()
}

type CompanionStruct_ChainsReadyResult_ struct {
}

var Companion_ChainsReadyResult_ = CompanionStruct_ChainsReadyResult_{}

type ChainsReadyResult_ChainsReadyResult struct {
	Blocks  _dafny.Map
	L1Heads _dafny.Map
}

func (ChainsReadyResult_ChainsReadyResult) isChainsReadyResult() {}

func (CompanionStruct_ChainsReadyResult_) Create_ChainsReadyResult_(Blocks _dafny.Map, L1Heads _dafny.Map) ChainsReadyResult {
	return ChainsReadyResult{ChainsReadyResult_ChainsReadyResult{Blocks, L1Heads}}
}

func (_this ChainsReadyResult) Is_ChainsReadyResult() bool {
	_, ok := _this.Get_().(ChainsReadyResult_ChainsReadyResult)
	return ok
}

func (CompanionStruct_ChainsReadyResult_) Default() ChainsReadyResult {
	return Companion_ChainsReadyResult_.Create_ChainsReadyResult_(_dafny.EmptyMap, _dafny.EmptyMap)
}

func (_this ChainsReadyResult) Dtor_blocks() _dafny.Map {
	return _this.Get_().(ChainsReadyResult_ChainsReadyResult).Blocks
}

func (_this ChainsReadyResult) Dtor_l1Heads() _dafny.Map {
	return _this.Get_().(ChainsReadyResult_ChainsReadyResult).L1Heads
}

func (_this ChainsReadyResult) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case ChainsReadyResult_ChainsReadyResult:
		{
			return "Types.ChainsReadyResult.ChainsReadyResult" + "(" + _dafny.String(data.Blocks) + ", " + _dafny.String(data.L1Heads) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this ChainsReadyResult) Equals(other ChainsReadyResult) bool {
	switch data1 := _this.Get_().(type) {
	case ChainsReadyResult_ChainsReadyResult:
		{
			data2, ok := other.Get_().(ChainsReadyResult_ChainsReadyResult)
			return ok && data1.Blocks.Equals(data2.Blocks) && data1.L1Heads.Equals(data2.L1Heads)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this ChainsReadyResult) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(ChainsReadyResult)
	return ok && _this.Equals(typed)
}

func Type_ChainsReadyResult_() _dafny.TypeDescriptor {
	return type_ChainsReadyResult_{}
}

type type_ChainsReadyResult_ struct {
}

func (_this type_ChainsReadyResult_) Default() interface{} {
	return Companion_ChainsReadyResult_.Default()
}

func (_this type_ChainsReadyResult_) String() string {
	return "Types.ChainsReadyResult"
}
func (_this ChainsReadyResult) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = ChainsReadyResult{}

// End of datatype ChainsReadyResult

// Definition of datatype StepOutput
type StepOutput struct {
	Data_StepOutput_
}

func (_this StepOutput) Get_() Data_StepOutput_ {
	return _this.Data_StepOutput_
}

type Data_StepOutput_ interface {
	isStepOutput()
}

type CompanionStruct_StepOutput_ struct {
}

var Companion_StepOutput_ = CompanionStruct_StepOutput_{}

type StepOutput_WaitOutput struct {
}

func (StepOutput_WaitOutput) isStepOutput() {}

func (CompanionStruct_StepOutput_) Create_WaitOutput_() StepOutput {
	return StepOutput{StepOutput_WaitOutput{}}
}

func (_this StepOutput) Is_WaitOutput() bool {
	_, ok := _this.Get_().(StepOutput_WaitOutput)
	return ok
}

type StepOutput_AdvanceOutput struct {
	Result Result
}

func (StepOutput_AdvanceOutput) isStepOutput() {}

func (CompanionStruct_StepOutput_) Create_AdvanceOutput_(Result Result) StepOutput {
	return StepOutput{StepOutput_AdvanceOutput{Result}}
}

func (_this StepOutput) Is_AdvanceOutput() bool {
	_, ok := _this.Get_().(StepOutput_AdvanceOutput)
	return ok
}

type StepOutput_InvalidateOutput struct {
	Result Result
}

func (StepOutput_InvalidateOutput) isStepOutput() {}

func (CompanionStruct_StepOutput_) Create_InvalidateOutput_(Result Result) StepOutput {
	return StepOutput{StepOutput_InvalidateOutput{Result}}
}

func (_this StepOutput) Is_InvalidateOutput() bool {
	_, ok := _this.Get_().(StepOutput_InvalidateOutput)
	return ok
}

type StepOutput_RewindOutput struct {
}

func (StepOutput_RewindOutput) isStepOutput() {}

func (CompanionStruct_StepOutput_) Create_RewindOutput_() StepOutput {
	return StepOutput{StepOutput_RewindOutput{}}
}

func (_this StepOutput) Is_RewindOutput() bool {
	_, ok := _this.Get_().(StepOutput_RewindOutput)
	return ok
}

func (CompanionStruct_StepOutput_) Default() StepOutput {
	return Companion_StepOutput_.Create_WaitOutput_()
}

func (_this StepOutput) Dtor_result() Result {
	switch data := _this.Get_().(type) {
	case StepOutput_AdvanceOutput:
		return data.Result
	default:
		return data.(StepOutput_InvalidateOutput).Result
	}
}

func (_this StepOutput) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case StepOutput_WaitOutput:
		{
			return "Types.StepOutput.WaitOutput"
		}
	case StepOutput_AdvanceOutput:
		{
			return "Types.StepOutput.AdvanceOutput" + "(" + _dafny.String(data.Result) + ")"
		}
	case StepOutput_InvalidateOutput:
		{
			return "Types.StepOutput.InvalidateOutput" + "(" + _dafny.String(data.Result) + ")"
		}
	case StepOutput_RewindOutput:
		{
			return "Types.StepOutput.RewindOutput"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this StepOutput) Equals(other StepOutput) bool {
	switch data1 := _this.Get_().(type) {
	case StepOutput_WaitOutput:
		{
			_, ok := other.Get_().(StepOutput_WaitOutput)
			return ok
		}
	case StepOutput_AdvanceOutput:
		{
			data2, ok := other.Get_().(StepOutput_AdvanceOutput)
			return ok && data1.Result.Equals(data2.Result)
		}
	case StepOutput_InvalidateOutput:
		{
			data2, ok := other.Get_().(StepOutput_InvalidateOutput)
			return ok && data1.Result.Equals(data2.Result)
		}
	case StepOutput_RewindOutput:
		{
			_, ok := other.Get_().(StepOutput_RewindOutput)
			return ok
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this StepOutput) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(StepOutput)
	return ok && _this.Equals(typed)
}

func Type_StepOutput_() _dafny.TypeDescriptor {
	return type_StepOutput_{}
}

type type_StepOutput_ struct {
}

func (_this type_StepOutput_) Default() interface{} {
	return Companion_StepOutput_.Default()
}

func (_this type_StepOutput_) String() string {
	return "Types.StepOutput"
}
func (_this StepOutput) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = StepOutput{}

// End of datatype StepOutput

// Definition of datatype ExecutingMessage
type ExecutingMessage struct {
	Data_ExecutingMessage_
}

func (_this ExecutingMessage) Get_() Data_ExecutingMessage_ {
	return _this.Data_ExecutingMessage_
}

type Data_ExecutingMessage_ interface {
	isExecutingMessage()
}

type CompanionStruct_ExecutingMessage_ struct {
}

var Companion_ExecutingMessage_ = CompanionStruct_ExecutingMessage_{}

type ExecutingMessage_ExecutingMessage struct {
	ChainID   _dafny.Int
	BlockNum  _dafny.Int
	LogIdx    _dafny.Int
	Timestamp _dafny.Int
	Checksum  _dafny.Int
}

func (ExecutingMessage_ExecutingMessage) isExecutingMessage() {}

func (CompanionStruct_ExecutingMessage_) Create_ExecutingMessage_(ChainID _dafny.Int, BlockNum _dafny.Int, LogIdx _dafny.Int, Timestamp _dafny.Int, Checksum _dafny.Int) ExecutingMessage {
	return ExecutingMessage{ExecutingMessage_ExecutingMessage{ChainID, BlockNum, LogIdx, Timestamp, Checksum}}
}

func (_this ExecutingMessage) Is_ExecutingMessage() bool {
	_, ok := _this.Get_().(ExecutingMessage_ExecutingMessage)
	return ok
}

func (CompanionStruct_ExecutingMessage_) Default() ExecutingMessage {
	return Companion_ExecutingMessage_.Create_ExecutingMessage_(_dafny.Zero, _dafny.Zero, _dafny.Zero, _dafny.Zero, _dafny.Zero)
}

func (_this ExecutingMessage) Dtor_chainID() _dafny.Int {
	return _this.Get_().(ExecutingMessage_ExecutingMessage).ChainID
}

func (_this ExecutingMessage) Dtor_blockNum() _dafny.Int {
	return _this.Get_().(ExecutingMessage_ExecutingMessage).BlockNum
}

func (_this ExecutingMessage) Dtor_logIdx() _dafny.Int {
	return _this.Get_().(ExecutingMessage_ExecutingMessage).LogIdx
}

func (_this ExecutingMessage) Dtor_timestamp() _dafny.Int {
	return _this.Get_().(ExecutingMessage_ExecutingMessage).Timestamp
}

func (_this ExecutingMessage) Dtor_checksum() _dafny.Int {
	return _this.Get_().(ExecutingMessage_ExecutingMessage).Checksum
}

func (_this ExecutingMessage) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case ExecutingMessage_ExecutingMessage:
		{
			return "Types.ExecutingMessage.ExecutingMessage" + "(" + _dafny.String(data.ChainID) + ", " + _dafny.String(data.BlockNum) + ", " + _dafny.String(data.LogIdx) + ", " + _dafny.String(data.Timestamp) + ", " + _dafny.String(data.Checksum) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this ExecutingMessage) Equals(other ExecutingMessage) bool {
	switch data1 := _this.Get_().(type) {
	case ExecutingMessage_ExecutingMessage:
		{
			data2, ok := other.Get_().(ExecutingMessage_ExecutingMessage)
			return ok && data1.ChainID.Cmp(data2.ChainID) == 0 && data1.BlockNum.Cmp(data2.BlockNum) == 0 && data1.LogIdx.Cmp(data2.LogIdx) == 0 && data1.Timestamp.Cmp(data2.Timestamp) == 0 && data1.Checksum.Cmp(data2.Checksum) == 0
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this ExecutingMessage) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(ExecutingMessage)
	return ok && _this.Equals(typed)
}

func Type_ExecutingMessage_() _dafny.TypeDescriptor {
	return type_ExecutingMessage_{}
}

type type_ExecutingMessage_ struct {
}

func (_this type_ExecutingMessage_) Default() interface{} {
	return Companion_ExecutingMessage_.Default()
}

func (_this type_ExecutingMessage_) String() string {
	return "Types.ExecutingMessage"
}
func (_this ExecutingMessage) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = ExecutingMessage{}

// End of datatype ExecutingMessage

// Definition of datatype ContainsQuery
type ContainsQuery struct {
	Data_ContainsQuery_
}

func (_this ContainsQuery) Get_() Data_ContainsQuery_ {
	return _this.Data_ContainsQuery_
}

type Data_ContainsQuery_ interface {
	isContainsQuery()
}

type CompanionStruct_ContainsQuery_ struct {
}

var Companion_ContainsQuery_ = CompanionStruct_ContainsQuery_{}

type ContainsQuery_ContainsQuery struct {
	BlockNum  _dafny.Int
	LogIdx    _dafny.Int
	Timestamp _dafny.Int
	Checksum  _dafny.Int
}

func (ContainsQuery_ContainsQuery) isContainsQuery() {}

func (CompanionStruct_ContainsQuery_) Create_ContainsQuery_(BlockNum _dafny.Int, LogIdx _dafny.Int, Timestamp _dafny.Int, Checksum _dafny.Int) ContainsQuery {
	return ContainsQuery{ContainsQuery_ContainsQuery{BlockNum, LogIdx, Timestamp, Checksum}}
}

func (_this ContainsQuery) Is_ContainsQuery() bool {
	_, ok := _this.Get_().(ContainsQuery_ContainsQuery)
	return ok
}

func (CompanionStruct_ContainsQuery_) Default() ContainsQuery {
	return Companion_ContainsQuery_.Create_ContainsQuery_(_dafny.Zero, _dafny.Zero, _dafny.Zero, _dafny.Zero)
}

func (_this ContainsQuery) Dtor_blockNum() _dafny.Int {
	return _this.Get_().(ContainsQuery_ContainsQuery).BlockNum
}

func (_this ContainsQuery) Dtor_logIdx() _dafny.Int {
	return _this.Get_().(ContainsQuery_ContainsQuery).LogIdx
}

func (_this ContainsQuery) Dtor_timestamp() _dafny.Int {
	return _this.Get_().(ContainsQuery_ContainsQuery).Timestamp
}

func (_this ContainsQuery) Dtor_checksum() _dafny.Int {
	return _this.Get_().(ContainsQuery_ContainsQuery).Checksum
}

func (_this ContainsQuery) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case ContainsQuery_ContainsQuery:
		{
			return "Types.ContainsQuery.ContainsQuery" + "(" + _dafny.String(data.BlockNum) + ", " + _dafny.String(data.LogIdx) + ", " + _dafny.String(data.Timestamp) + ", " + _dafny.String(data.Checksum) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this ContainsQuery) Equals(other ContainsQuery) bool {
	switch data1 := _this.Get_().(type) {
	case ContainsQuery_ContainsQuery:
		{
			data2, ok := other.Get_().(ContainsQuery_ContainsQuery)
			return ok && data1.BlockNum.Cmp(data2.BlockNum) == 0 && data1.LogIdx.Cmp(data2.LogIdx) == 0 && data1.Timestamp.Cmp(data2.Timestamp) == 0 && data1.Checksum.Cmp(data2.Checksum) == 0
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this ContainsQuery) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(ContainsQuery)
	return ok && _this.Equals(typed)
}

func Type_ContainsQuery_() _dafny.TypeDescriptor {
	return type_ContainsQuery_{}
}

type type_ContainsQuery_ struct {
}

func (_this type_ContainsQuery_) Default() interface{} {
	return Companion_ContainsQuery_.Default()
}

func (_this type_ContainsQuery_) String() string {
	return "Types.ContainsQuery"
}
func (_this ContainsQuery) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = ContainsQuery{}

// End of datatype ContainsQuery

// Definition of datatype Log
type Log struct {
	Data_Log_
}

func (_this Log) Get_() Data_Log_ {
	return _this.Data_Log_
}

type Data_Log_ interface {
	isLog()
}

type CompanionStruct_Log_ struct {
}

var Companion_Log_ = CompanionStruct_Log_{}

type Log_Log struct {
	Data     _dafny.Int
	Checksum _dafny.Int
}

func (Log_Log) isLog() {}

func (CompanionStruct_Log_) Create_Log_(Data _dafny.Int, Checksum _dafny.Int) Log {
	return Log{Log_Log{Data, Checksum}}
}

func (_this Log) Is_Log() bool {
	_, ok := _this.Get_().(Log_Log)
	return ok
}

func (CompanionStruct_Log_) Default() Log {
	return Companion_Log_.Create_Log_(_dafny.Zero, _dafny.Zero)
}

func (_this Log) Dtor_data() _dafny.Int {
	return _this.Get_().(Log_Log).Data
}

func (_this Log) Dtor_checksum() _dafny.Int {
	return _this.Get_().(Log_Log).Checksum
}

func (_this Log) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case Log_Log:
		{
			return "Types.Log.Log" + "(" + _dafny.String(data.Data) + ", " + _dafny.String(data.Checksum) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this Log) Equals(other Log) bool {
	switch data1 := _this.Get_().(type) {
	case Log_Log:
		{
			data2, ok := other.Get_().(Log_Log)
			return ok && data1.Data.Cmp(data2.Data) == 0 && data1.Checksum.Cmp(data2.Checksum) == 0
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this Log) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(Log)
	return ok && _this.Equals(typed)
}

func Type_Log_() _dafny.TypeDescriptor {
	return type_Log_{}
}

type type_Log_ struct {
}

func (_this type_Log_) Default() interface{} {
	return Companion_Log_.Default()
}

func (_this type_Log_) String() string {
	return "Types.Log"
}
func (_this Log) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = Log{}

// End of datatype Log

// Definition of datatype BlockLogs
type BlockLogs struct {
	Data_BlockLogs_
}

func (_this BlockLogs) Get_() Data_BlockLogs_ {
	return _this.Data_BlockLogs_
}

type Data_BlockLogs_ interface {
	isBlockLogs()
}

type CompanionStruct_BlockLogs_ struct {
}

var Companion_BlockLogs_ = CompanionStruct_BlockLogs_{}

type BlockLogs_BlockLogs struct {
	FullLogs _dafny.Sequence
	ExecMsgs _dafny.Map
}

func (BlockLogs_BlockLogs) isBlockLogs() {}

func (CompanionStruct_BlockLogs_) Create_BlockLogs_(FullLogs _dafny.Sequence, ExecMsgs _dafny.Map) BlockLogs {
	return BlockLogs{BlockLogs_BlockLogs{FullLogs, ExecMsgs}}
}

func (_this BlockLogs) Is_BlockLogs() bool {
	_, ok := _this.Get_().(BlockLogs_BlockLogs)
	return ok
}

func (CompanionStruct_BlockLogs_) Default() BlockLogs {
	return Companion_BlockLogs_.Create_BlockLogs_(_dafny.EmptySeq, _dafny.EmptyMap)
}

func (_this BlockLogs) Dtor_fullLogs() _dafny.Sequence {
	return _this.Get_().(BlockLogs_BlockLogs).FullLogs
}

func (_this BlockLogs) Dtor_execMsgs() _dafny.Map {
	return _this.Get_().(BlockLogs_BlockLogs).ExecMsgs
}

func (_this BlockLogs) String() string {
	switch data := _this.Get_().(type) {
	case nil:
		return "null"
	case BlockLogs_BlockLogs:
		{
			return "Types.BlockLogs.BlockLogs" + "(" + _dafny.String(data.FullLogs) + ", " + _dafny.String(data.ExecMsgs) + ")"
		}
	default:
		{
			return "<unexpected>"
		}
	}
}

func (_this BlockLogs) Equals(other BlockLogs) bool {
	switch data1 := _this.Get_().(type) {
	case BlockLogs_BlockLogs:
		{
			data2, ok := other.Get_().(BlockLogs_BlockLogs)
			return ok && data1.FullLogs.Equals(data2.FullLogs) && data1.ExecMsgs.Equals(data2.ExecMsgs)
		}
	default:
		{
			return false // unexpected
		}
	}
}

func (_this BlockLogs) EqualsGeneric(other interface{}) bool {
	typed, ok := other.(BlockLogs)
	return ok && _this.Equals(typed)
}

func Type_BlockLogs_() _dafny.TypeDescriptor {
	return type_BlockLogs_{}
}

type type_BlockLogs_ struct {
}

func (_this type_BlockLogs_) Default() interface{} {
	return Companion_BlockLogs_.Default()
}

func (_this type_BlockLogs_) String() string {
	return "Types.BlockLogs"
}
func (_this BlockLogs) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = BlockLogs{}

// End of datatype BlockLogs
