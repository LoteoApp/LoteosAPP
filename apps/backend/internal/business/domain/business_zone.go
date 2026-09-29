package domain

import "time"

// BusinessLocation is the time zone business dates are computed in.
// Argentina has no daylight saving time, so a fixed UTC-3 offset is exact and
// doesn't depend on tzdata, which the distroless production image lacks.
var BusinessLocation = time.FixedZone("UTC-3", -3*60*60)
