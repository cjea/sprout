package mvp

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PassageRepairSessionStore interface {
	SavePassageRepairSession(*PassageRepairSession) error
	LoadPassageRepairSession() (*PassageRepairSession, error)
}

type JSONPassageRepairSessionStore struct {
	Path string
}

func (s JSONPassageRepairSessionStore) SavePassageRepairSession(session *PassageRepairSession) error {
	if session == nil {
		return ErrPassageRepairRevisionInvalid
	}
	if err := session.Current.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, data, 0o644)
}

func (s JSONPassageRepairSessionStore) LoadPassageRepairSession() (*PassageRepairSession, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var session PassageRepairSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	if err := session.Current.Validate(); err != nil {
		return nil, err
	}
	return &session, nil
}
