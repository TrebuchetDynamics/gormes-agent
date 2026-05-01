package apiserver

type LogStore struct{}

func NewLogStore(retentionDays int) *LogStore { return &LogStore{} }
