package models

type Drive struct {
	Name      string `json:"name"`
	ID        string `json:"id"`
	Enabled   bool   `json:"enabled"`
	AutoMount bool   `json:"auto_mount"`
}
