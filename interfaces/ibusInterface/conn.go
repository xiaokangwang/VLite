package ibusInterface

type ConnBoostMode struct {
	Enable         bool
	TimeoutAtLeast int
}

type ConnReHandshake struct {
	Fire            bool
	FullReHandshake bool
}
