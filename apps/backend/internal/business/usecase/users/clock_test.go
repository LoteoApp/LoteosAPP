package users

import "time"

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

type mutableClock struct{ now time.Time }

func (clock *mutableClock) Now() time.Time { return clock.now }
