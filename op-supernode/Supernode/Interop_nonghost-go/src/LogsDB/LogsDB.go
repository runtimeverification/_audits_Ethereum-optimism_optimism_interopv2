// Package LogsDB
// Dafny module LogsDB compiled into Go

package LogsDB

import (
	m_ChainContainer "ChainContainer"
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

type Dummy__ struct{}

// Definition of class LogsDB
type LogsDB struct {
	dummy byte
}

func New_LogsDB_() *LogsDB {
	_this := LogsDB{}

	return &_this
}

type CompanionStruct_LogsDB_ struct {
}

var Companion_LogsDB_ = CompanionStruct_LogsDB_{}

func (_this *LogsDB) Equals(other *LogsDB) bool {
	return _this == other
}

func (_this *LogsDB) EqualsGeneric(x interface{}) bool {
	other, ok := x.(*LogsDB)
	return ok && _this.Equals(other)
}

func (*LogsDB) String() string {
	return "LogsDB.LogsDB"
}

func Type_LogsDB_() _dafny.TypeDescriptor {
	return type_LogsDB_{}
}

type type_LogsDB_ struct {
}

func (_this type_LogsDB_) Default() interface{} {
	return (*LogsDB)(nil)
}

func (_this type_LogsDB_) String() string {
	return "LogsDB.LogsDB"
}
func (_this *LogsDB) ParentTraits_() []*_dafny.TraitID {
	return [](*_dafny.TraitID){}
}

var _ _dafny.TraitOffspring = &LogsDB{}

func (_this *LogsDB) Ctor__() {
	{
	}
}
func (_this *LogsDB) Clear() {
	{
	}
}
func (_this *LogsDB) Rewind(newHead m_Types.BlockID) {
	{
	}
}
func (_this *LogsDB) FirstSealedBlock() m_Types.Option {
	{
		return m_Types.Companion_Option_.Create_None_()
	}
}
func (_this *LogsDB) LatestSealedBlock() m_Types.Option {
	{
		return m_Types.Companion_Option_.Create_None_()
	}
}
func (_this *LogsDB) FindSealedBlock(number _dafny.Int) m_Types.Option {
	{
		return m_Types.Companion_Option_.Create_None_()
	}
}
func (_this *LogsDB) BlockLogs(blockNum _dafny.Int) m_Types.BlockLogs {
	{
		return m_Types.Companion_BlockLogs_.Create_BlockLogs_(_dafny.SeqOf(), _dafny.NewMapBuilder().ToMap())
	}
}
func (_this *LogsDB) OpenBlock(blockNum _dafny.Int) (m_Types.BlockSeal, _dafny.Map) {
	{
		var ref m_Types.BlockSeal = m_Types.Companion_BlockSeal_.Default()
		_ = ref
		var execMsgs _dafny.Map = _dafny.EmptyMap
		_ = execMsgs
		return ref, execMsgs
	}
}
func (_this *LogsDB) Contains(query m_Types.ContainsQuery) bool {
	{
		return false
	}
}

// End of class LogsDB
