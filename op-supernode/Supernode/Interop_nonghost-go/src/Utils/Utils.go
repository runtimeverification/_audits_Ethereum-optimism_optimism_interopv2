// Package Utils
// Dafny module Utils compiled into Go

package Utils

import (
	m__System "System_"
	m_Types "Types"
	_dafny "dafny"
	os "os"
)

var _ = os.Args
var _ _dafny.Dummy__
var _ m__System.Dummy__
var _ m_Types.Dummy__

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
	return "Utils.Default__"
}
func (_this *Default__) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = &Default__{}

func (_static *CompanionStruct_Default___) Max(s _dafny.Set) _dafny.Int {
	var _pat_let_tv0 = s
	_ = _pat_let_tv0
	var _pat_let_tv1 = s
	_ = _pat_let_tv1
	return func(_let_dummy_0 int) _dafny.Int {
		var _0_x _dafny.Int = _dafny.Zero
		_ = _0_x
		{
			for _iter1 := _dafny.Iterate((s).Elements()); ; {
				_assign_such_that_0, _ok1 := _iter1()
				if !_ok1 {
					break
				}
				_0_x = interface{}(_assign_such_that_0).(_dafny.Int)
				if ((s).Contains(_0_x)) && (_dafny.Quantifier((s).Elements(), true, func(_forall_var_0 _dafny.Int) bool {
					var _1_y _dafny.Int
					_1_y = interface{}(_forall_var_0).(_dafny.Int)
					return !((s).Contains(_1_y)) || ((_0_x).Cmp(_1_y) <= 0)
				})) {
					goto L_ASSIGN_SUCH_THAT_0
				}
			}
			panic("assign-such-that search produced no value")
			goto L_ASSIGN_SUCH_THAT_0
		}
	L_ASSIGN_SUCH_THAT_0:
		return (func() _dafny.Int {
			if (_pat_let_tv0).Equals(_dafny.SetOf(_0_x)) {
				return _0_x
			}
			return func(_pat_let1_0 _dafny.Set) _dafny.Int {
				return func(_2_rest _dafny.Set) _dafny.Int {
					return func(_pat_let2_0 _dafny.Int) _dafny.Int {
						return func(_3_maxRest _dafny.Int) _dafny.Int {
							return (func() _dafny.Int {
								if (_0_x).Cmp(_3_maxRest) >= 0 {
									return _0_x
								}
								return _3_maxRest
							})()
						}(_pat_let2_0)
					}(Companion_Default___.Max(_2_rest))
				}(_pat_let1_0)
			}((_pat_let_tv1).Difference(_dafny.SetOf(_0_x)))
		})()
	}(0)
}
func (_static *CompanionStruct_Default___) MaxKey(m _dafny.Map) _dafny.Int {
	return Companion_Default___.Max((m).Keys())
}
func (_static *CompanionStruct_Default___) Sequential(m _dafny.Map) bool {
	return (((m).Cardinality()).Sign() == 0) || (_dafny.Quantifier((m).Keys().Elements(), true, func(_forall_var_0 _dafny.Int) bool {
		var _0_k _dafny.Int
		_0_k = interface{}(_forall_var_0).(_dafny.Int)
		return !(((m).Contains(_0_k)) && ((_0_k).Cmp(Companion_Default___.MaxKey(m)) < 0)) || ((m).Contains((_0_k).Plus(_dafny.One)))
	}))
}

// End of class Default__
