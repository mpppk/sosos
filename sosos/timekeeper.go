package sosos

import (
	"time"

	"github.com/mpppk/sosos/etc"
)

type TimeKeeper struct {
	sleepSec           int64
	orgRemindSeconds   []int64
	remindSeconds      []int64
	suspendMinutes     []int64
	commandExecuteTime time.Time
	remainSec          int64
}

func NewTimeKeeper(sleepSec int64, remindSeconds []int64, suspendMinutes []int64) *TimeKeeper {
	commandExecuteTime := time.Now().Add(time.Duration(sleepSec) * time.Second)
	remainSec := commandExecuteTime.Unix() - time.Now().Unix()

	t := &TimeKeeper{
		sleepSec:           sleepSec,
		orgRemindSeconds:   remindSeconds,
		remindSeconds:      remindSeconds,
		suspendMinutes:     suspendMinutes,
		commandExecuteTime: commandExecuteTime,
		remainSec:          remainSec,
	}

	t.UpdateRemindSeconds()
	return t
}

func (t *TimeKeeper) UpdateRemindSeconds() {
	t.UpdateRemainSec()
	t.remindSeconds = t.orgRemindSeconds
	for _, second := range t.orgRemindSeconds {
		if second > t.remainSec {
			t.remindSeconds = etc.Remove(t.remindSeconds, second)
		}
	}
}

func (t *TimeKeeper) SuspendCommandExecuteTime(suspendSec int) {
	t.commandExecuteTime = t.commandExecuteTime.Add(time.Duration(suspendSec) * time.Second)
}

func (t *TimeKeeper) UpdateRemainSec() {
	t.remainSec = t.commandExecuteTime.Unix() - time.Now().Unix()
}

func (t *TimeKeeper) GetNewRemind() (int64, bool) {
	var remindSecond int64
	for _, second := range t.remindSeconds {
		if second > t.remainSec {
			t.remindSeconds = etc.Remove(t.remindSeconds, second)
			remindSecond = second
		}
	}
	return remindSecond, remindSecond != 0
}
