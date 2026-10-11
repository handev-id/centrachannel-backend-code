package note

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type noteRepository struct{}

func NewNoteRepository() NoteRepository {
	return &noteRepository{}
}

func scanNote(row interface{ Scan(...interface{}) error }) (*Note, error) {
	var n Note
	var date sql.NullTime
	var userID uuid.NullUUID
	if err := row.Scan(&n.ID, &n.TenantID, &n.Text, &date, &n.ConversationID, &userID, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("note not found")
		}
		return nil, err
	}
	if date.Valid {
		n.Date = &date.Time
	}
	if userID.Valid {
		n.UserID = &userID.UUID
	}
	return &n, nil
}

func (r *noteRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Note, error) {
	return scanNote(q.QueryRowContext(ctx, `SELECT id, tenant_id, text, date, conversation_id, user_id, created_at, updated_at FROM notes WHERE id=$1 AND tenant_id=$2`, id, tenantID))
}

func (r *noteRepository) ListByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, text, date, conversation_id, user_id, created_at, updated_at FROM notes WHERE conversation_id = $1 AND tenant_id = $2 ORDER BY created_at DESC`, conversationID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *n)
	}
	return notes, rows.Err()
}

func (r *noteRepository) Create(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRowContext(ctx, `INSERT INTO notes (tenant_id, text, date, conversation_id, user_id, created_at, updated_at)
		SELECT $1,$2,$3,c.id,$5,$6,$7 FROM conversations c
		WHERE c.id=$4 AND c.tenant_id=$1
		AND ($5::uuid IS NULL OR EXISTS (SELECT 1 FROM users u WHERE u.id=$5 AND u.tenant_id=$1 AND u.deleted_at IS NULL))
		RETURNING id`, note.TenantID, note.Text, note.Date, note.ConversationID, note.UserID, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *noteRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error {
	result, err := q.ExecContext(ctx, `UPDATE notes SET text=$1, date=$2, updated_at=$3 WHERE id=$4 AND tenant_id=$5`, note.Text, note.Date, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("note not found")
	}
	return nil
}

func (r *noteRepository) Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	result, err := q.ExecContext(ctx, `DELETE FROM notes WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("note not found")
	}
	return nil
}
