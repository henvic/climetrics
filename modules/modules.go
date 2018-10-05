package modules

import (
	// diagnostics routes
	_ "github.com/henvic/climetrics/diagnostics/handlers"

	// metrics routes
	_ "github.com/henvic/climetrics/metrics/handlers"

	// users routes
	_ "github.com/henvic/climetrics/users/handlers"

	// auth routes
	_ "github.com/henvic/climetrics/auth/handlers"
)
