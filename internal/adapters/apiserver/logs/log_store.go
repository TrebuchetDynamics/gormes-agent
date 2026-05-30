package logs

type Store struct{}

func NewStore(retentionDays int) *Store { return &Store{} }
