package tx

import (
	"context"

	"github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/avito-tech/go-transaction-manager/trm/manager"
	"github.com/avito-tech/go-transaction-manager/trm/settings"
	"github.com/jackc/pgx/v5"
)

// Manager инкапсулирует логику управления транзакциями.
type Manager struct {
	internal *manager.Manager
}

// New создаёт новый менеджер транзакций.
func New(db pgxv5.Transactional) *Manager {
	return &Manager{
		internal: manager.Must(pgxv5.NewDefaultFactory(db)),
	}
}

func (m *Manager) execWithIsoLevel(
	ctx context.Context,
	level pgx.TxIsoLevel,
	fn func(ctx context.Context) error,
) error {
	txSettings := pgxv5.MustSettings(
		settings.Must(),
		pgxv5.WithTxOptions(pgx.TxOptions{IsoLevel: level}),
	)
	return m.internal.DoWithSettings(ctx, txSettings, fn)
}

func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.execWithIsoLevel(ctx, pgx.Serializable, fn)
}
