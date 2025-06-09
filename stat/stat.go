package stat

import (
	"time"
)

type ExecutionTache struct {
	Duree   time.Duration
	Erreur  bool
	Execute bool
}

type Stat interface {
	Put(nom string, valeur ExecutionTache)
	Keys() []string
	Get(nom string) ExecutionTache
}

type StatTache struct {
	elements map[string]ExecutionTache
	cles     []string
}

func CreateStat() Stat {
	return &StatTache{elements: make(map[string]ExecutionTache), cles: make([]string, 0)}
}

func CreateExecutionTache() ExecutionTache {
	return ExecutionTache{Duree: 0, Erreur: false, Execute: false}
}

func (s *StatTache) Put(nom string, valeur ExecutionTache) {
	s.elements[nom] = valeur
	trouve := false
	for _, t := range s.cles {
		if t == nom {
			trouve = true
		}
	}
	if !trouve {
		s.cles = append(s.cles, nom)
	}
}

func (s *StatTache) Keys() []string {
	return s.cles
}

func (s *StatTache) Get(nom string) ExecutionTache {
	return s.elements[nom]
}
