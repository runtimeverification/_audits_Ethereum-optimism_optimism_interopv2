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
	return "_module.VerifiedDB"
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
			for _iter1 := _dafny.Iterate((_this.Db).Keys().Elements()); ; {
				_compr_0, _ok1 := _iter1()
				if !_ok1 {
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
