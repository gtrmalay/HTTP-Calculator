package storage

import (
	"sync"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
)

var (
	Expressions = make(map[string]*models.Expression)
	Tasks       = make(map[string]*models.Task)
	TaskQueue   = make([]string, 0)
	Mu          sync.Mutex
)
