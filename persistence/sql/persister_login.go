package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/ory/kratos/corp"

	"github.com/gofrs/uuid"

	"github.com/ory/x/sqlcon"

	"github.com/ory/kratos/selfservice/flow/login"
)

var _ login.FlowPersister = new(Persister)

func (p *Persister) CreateLoginFlow(ctx context.Context, r *login.Flow) error {
	r.NID = corp.ContextualizeNID(ctx, p.nid)
	r.EnsureInternalContext()
	return p.GetConnection(ctx).Create(r)
}

func (p *Persister) UpdateLoginFlow(ctx context.Context, r *login.Flow) error {
	r.EnsureInternalContext()
	cp := *r
	cp.NID = corp.ContextualizeNID(ctx, p.nid)
	return p.update(ctx, cp)
}

func (p *Persister) GetLoginFlow(ctx context.Context, id uuid.UUID) (*login.Flow, error) {
	conn := p.GetConnection(ctx)

	var r login.Flow
	if err := conn.Where("id = ? AND nid = ?", id, corp.ContextualizeNID(ctx, p.nid)).First(&r); err != nil {
		return nil, sqlcon.HandleError(err)
	}

	return &r, nil
}

func (p *Persister) ForceLoginFlow(ctx context.Context, id uuid.UUID) error {
	return p.Transaction(ctx, func(ctx context.Context, tx *pop.Connection) error {
		lr, err := p.GetLoginFlow(ctx, id)
		if err != nil {
			return err
		}

		lr.Refresh = true
		return tx.Save(lr, "nid")
	})
}

func (p *Persister) DeleteExpiredLoginFlows(ctx context.Context, expiresAt time.Time, limit, batch int) error {
	for ok := true; ok; ok = batch <= limit {
		limit -= batch
		// #nosec G201
		count, err := p.GetConnection(ctx).RawQuery(fmt.Sprintf(
			"DELETE FROM `%s` WHERE `expires_at` <= ? LIMIT ?",
			new(login.Flow).TableName(ctx),
		),
			expiresAt,
			batch,
		).ExecWithCount()
		if err != nil {
			return sqlcon.HandleError(err)
		}

		if count == 0 {
			break
		}
	}
	return nil
}
