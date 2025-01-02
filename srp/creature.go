package srp

type Creature struct {
	ID          int64
	Name        string
	Description string
}

type CreatureLookupResult struct {
	ResultFound bool
	Creature    Creature
}
