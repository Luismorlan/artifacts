package model

type SeedState struct {
	UserSeedState   *UserSeedState     `json:"userSeedState"`
	ColumnSeedState []*ColumnSeedState `json:"columnSeedState"`
}
