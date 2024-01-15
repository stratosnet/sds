package core

import (
	"github.com/stratosnet/sds/framework/utils"
)

type onStartLogFunc func(*Server)

type volRecOpts struct {
	onStartLog       onStartLogFunc
	logInbound       bool
	logOutbound      bool
	logRead          bool
	logWrite         bool
	logAll           bool
	allFlow          int64 //including read flow & write flow
	allAtom          *utils.AtomicInt64
	readFlow         int64 //not used for now
	readAtom         *utils.AtomicInt64
	writeFlow        int64 //not used for now
	writeAtom        *utils.AtomicInt64
	secondReadFlowA  int64 // will be reset to 0 every second by logFunc() job
	secondReadAtomA  *utils.AtomicInt64
	secondWriteFlowA int64 // will be reset to 0 every second by logFunc() job
	secondWriteAtomA *utils.AtomicInt64
	secondReadFlowB  int64 //for monitor use, will be refreshed to the value of secondReadFlowA before secondReadFlowA is reset to 0 every second by logFunc() job
	secondReadAtomB  *utils.AtomicInt64
	secondWriteFlowB int64 //for monitor use, will be refreshed to the value of secondWriteFlowA before secondWriteFlowA is reset to 0 every second by logFunc() job
	secondWriteAtomB *utils.AtomicInt64
	inbound          int64              // for traffic log
	inboundAtomic    *utils.AtomicInt64 // for traffic log
	outbound         int64              // for traffic log
	outboundAtomic   *utils.AtomicInt64 // for traffic log
}

type ServerVolRecOption func(*volRecOpts)

// LogAllOption
func LogAllOption(logOpen bool) ServerVolRecOption {
	return func(o *volRecOpts) {
		o.logAll = logOpen
		o.allAtom = utils.CreateAtomicInt64(0)
	}
}

// LogReadOption
func LogReadOption(logOpen bool) ServerVolRecOption {
	return func(o *volRecOpts) {
		o.logRead = logOpen
		o.readAtom = utils.CreateAtomicInt64(0)
		o.secondReadAtomA = utils.CreateAtomicInt64(0)
		o.secondReadAtomB = utils.CreateAtomicInt64(0)
	}
}

// OnWriteOption
func OnWriteOption(logOpen bool) ServerVolRecOption {
	return func(o *volRecOpts) {
		o.logWrite = logOpen
		o.writeAtom = utils.CreateAtomicInt64(0)
		o.secondWriteAtomA = utils.CreateAtomicInt64(0)
		o.secondWriteAtomB = utils.CreateAtomicInt64(0)
	}
}

// LogInboundOption
func LogInboundOption(logOpen bool) ServerVolRecOption {
	return func(o *volRecOpts) {
		o.logInbound = logOpen
		o.inboundAtomic = utils.CreateAtomicInt64(0)
	}
}

// LogOutboundOption
func LogOutboundOption(logOpen bool) ServerVolRecOption {
	return func(o *volRecOpts) {
		o.logOutbound = logOpen
		o.outboundAtomic = utils.CreateAtomicInt64(0)
	}
}
