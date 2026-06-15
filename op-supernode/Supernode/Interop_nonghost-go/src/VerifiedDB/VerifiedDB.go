// Package VerifiedDB
// Dafny module VerifiedDB compiled into Go

package VerifiedDB

import (
	m__System "System_"
	m_Types "Types"
	m_Utils "Utils"
	_dafny "dafny"
	os "os"
)

var _ = os.Args
var _ _dafny.Dummy__
var _ m__System.Dummy__
var _ m_Types.Dummy__
var _ m_Utils.Dummy__

type Dummy__ struct{}

// Definition of class VerifiedDB
type VerifiedDB struct {
	Db                _dafny.Map
	Go__lastTimestamp m_Types.Option
	PendingTransition m_Types.Option
}

func New_VerifiedDB_() *VerifiedDB {
	_this := VerifiedDB{}

	_this.Db = _dafny.EmptyMap
	_this.Go__lastTimestamp = m_Types.Companion_Option_.Default()
	_this.PendingTransition = m_Types.Companion_Option_.Default()
	return &_this
}

type CompanionStruct_VerifiedDB_ struct {
}

var Companion_VerifiedDB_ = CompanionStruct_VerifiedDB_{}

func (_this *VerifiedDB) Equals(other *VerifiedDB) bool {
	return _this == other
}

func (_this *VerifiedDB) EqualsGeneric(x interface{}) bool {
	other, ok := x.(*VerifiedDB)
	return ok && _this.Equals(other)
}

func (*VerifiedDB) String() string {
	return "VerifiedDB.VerifiedDB"
}

func Type_VerifiedDB_() _dafny.TypeDescriptor {
	return type_VerifiedDB_{}
}

type type_VerifiedDB_ struct {
}

func (_this type_VerifiedDB_) Default() interface{} {
	return (*VerifiedDB)(nil)
}

func (_this type_VerifiedDB_) String() string {
	return "VerifiedDB.VerifiedDB"
}
func (_this *VerifiedDB) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = &VerifiedDB{}

func (_this *VerifiedDB) Valid() bool {
	{
		return (((m_Utils.Companion_Default___.Sequential(_this.Db)) && (_dafny.Quantifier((_this.Db).Keys().Elements(), true, func(_forall_var_0 _dafny.Int) bool {
			var _0_ts _dafny.Int
			_0_ts = interface{}(_forall_var_0).(_dafny.Int)
			return !((_this.Db).Contains(_0_ts)) || ((((_this.Db).Get(_0_ts).(m_Types.VerifiedResult)).Dtor_timestamp()).Cmp(_0_ts) == 0)
		}))) && ((_this.Go__lastTimestamp).Equals((func() m_Types.Option {
			if ((_this.Db).Cardinality()).Sign() == 0 {
				return m_Types.Companion_Option_.Create_None_()
			}
			return m_Types.Companion_Option_.Create_Some_(m_Utils.Companion_Default___.MaxKey(_this.Db))
		})()))) && (_dafny.Quantifier((_this.Db).Keys().Elements(), true, func(_forall_var_1 _dafny.Int) bool {
			var _1_t1 _dafny.Int
			_1_t1 = interface{}(_forall_var_1).(_dafny.Int)
			return _dafny.Quantifier((_this.Db).Keys().Elements(), true, func(_forall_var_2 _dafny.Int) bool {
				var _2_t2 _dafny.Int
				_2_t2 = interface{}(_forall_var_2).(_dafny.Int)
				return _dafny.Quantifier((((_this.Db).Get(_1_t1).(m_Types.VerifiedResult)).Dtor_l2Heads()).Keys().Elements(), true, func(_forall_var_3 _dafny.Int) bool {
					var _3_cid _dafny.Int
					_3_cid = interface{}(_forall_var_3).(_dafny.Int)
					return !((((((_this.Db).Contains(_1_t1)) && ((_this.Db).Contains(_2_t2))) && ((_1_t1).Cmp(_2_t2) <= 0)) && ((((_this.Db).Get(_1_t1).(m_Types.VerifiedResult)).Dtor_l2Heads()).Contains(_3_cid))) && ((((_this.Db).Get(_2_t2).(m_Types.VerifiedResult)).Dtor_l2Heads()).Contains(_3_cid))) || ((((((_this.Db).Get(_1_t1).(m_Types.VerifiedResult)).Dtor_l2Heads()).Get(_3_cid).(m_Types.BlockID)).Dtor_number()).Cmp(((((_this.Db).Get(_2_t2).(m_Types.VerifiedResult)).Dtor_l2Heads()).Get(_3_cid).(m_Types.BlockID)).Dtor_number()) <= 0)
				})
			})
		}))
	}
}
func (_this *VerifiedDB) Ctor__() {
	{
		(_this).Db = _dafny.NewMapBuilder().ToMap()
		(_this).Go__lastTimestamp = m_Types.Companion_Option_.Create_None_()
		(_this).PendingTransition = m_Types.Companion_Option_.Create_None_()
	}
}
func (_this *VerifiedDB) Commit(result m_Types.VerifiedResult) {
	{
		(_this).Db = (_this.Db).Update((result).Dtor_timestamp(), result)
		(_this).Go__lastTimestamp = m_Types.Companion_Option_.Create_Some_((result).Dtor_timestamp())
	}
}
func (_this *VerifiedDB) Get(ts _dafny.Int) m_Types.VerifiedResult {
	{
		return (_this.Db).Get(ts).(m_Types.VerifiedResult)
	}
}
func (_this *VerifiedDB) Has(ts _dafny.Int) bool {
	{
		return (_this.Db).Contains(ts)
	}
}
func (_this *VerifiedDB) LastTimestamp() m_Types.Option {
	{
		return _this.Go__lastTimestamp
	}
}
func (_this *VerifiedDB) Rewind(timestamp _dafny.Int) bool {
	{
		var deleted bool = false
		_ = deleted
		var _0_newDb _dafny.Map
		_ = _0_newDb
		_0_newDb = func() _dafny.Map {
			var _coll0 = _dafny.NewMapBuilder()
			_ = _coll0
			for _iter2 := _dafny.Iterate((_this.Db).Keys().Elements()); ; {
				_compr_0, _ok2 := _iter2()
				if !_ok2 {
					break
				}
				var _1_k _dafny.Int
				_1_k = interface{}(_compr_0).(_dafny.Int)
				if ((_this.Db).Contains(_1_k)) && ((_1_k).Cmp(timestamp) < 0) {
					_coll0.Add(_1_k, (_this.Db).Get(_1_k).(m_Types.VerifiedResult))
				}
			}
			return _coll0.ToMap()
		}()
		deleted = ((_0_newDb).Cardinality()).Cmp((_this.Db).Cardinality()) < 0
		if ((_0_newDb).Cardinality()).Sign() == 0 {
			(_this).Go__lastTimestamp = m_Types.Companion_Option_.Create_None_()
		} else {
			var _2_oldMax _dafny.Int
			_ = _2_oldMax
			_2_oldMax = (_this.Go__lastTimestamp).Dtor_value().(_dafny.Int)
			(_this).Go__lastTimestamp = m_Types.Companion_Option_.Create_Some_((func() _dafny.Int {
				if (_2_oldMax).Cmp(timestamp) < 0 {
					return _2_oldMax
				}
				return (timestamp).Minus(_dafny.One)
			})())
		}
		(_this).Db = _0_newDb
		return deleted
	}
}
func (_this *VerifiedDB) SetPendingTransition(pending m_Types.PendingTransition) {
	{
		(_this).PendingTransition = m_Types.Companion_Option_.Create_Some_(pending)
	}
}
func (_this *VerifiedDB) GetPendingTransition() m_Types.Option {
	{
		return _this.PendingTransition
	}
}
func (_this *VerifiedDB) ClearPendingTransition() {
	{
		(_this).PendingTransition = m_Types.Companion_Option_.Create_None_()
	}
}

// End of class VerifiedDB
