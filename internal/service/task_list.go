package service

import (
	"time"
)

type Template struct {
	Name           string          `json:"name"`
	CacheNamespace string          `json:"cacheNamespace"`
	CacheScope     string          `json:"cacheScope"`
	Groups         []GroupTemplate `json:"groups"`
}

type GroupTemplate struct {
	Execution   string   `json:"execution"`
	FinalPolicy string   `json:"finalPolicy"`
	Tasks       []string `json:"tasks"`
}

type TaskList struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	State          State     `json:"state"`
	Groups         []Group   `json:"groups"`
	Request        []byte    `json:"request"`
	CacheNamespace string    `json:"cacheNamespace"`
	CacheScope     string    `json:"cacheScope"`
	CreatedAt      time.Time `json:"createdAt"`
	StartedAt      time.Time `json:"startedAt"`
	FinishedAt     time.Time `json:"finishedAt"`
}

type Group struct {
	ID          string   `json:"id"`
	Execution   string   `json:"execution"`
	Tasks       []string `json:"tasks"`
	State       State    `json:"state"`
	Request     []byte   `json:"request"`
	FinalPolicy string   `json:"finalPolicy"`
}
