package note

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type noteRepository struct{}

func NewNoteRepository() NoteRepository {
	return &noteRepository{}
}

func (r *noteRepository) ListByConversation(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, text, date, conversation_id, user_id, created_at, updated_at FROM notes WHERE conversation_id = $1 AND tenant_id = $2 ORDER BY created_at DESC`, conversationID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		var date sql.NullTime
		var userID sql.NullInt64
		if err := rows.Scan(&n.ID, &n.TenantID, &n.Text, &date, &n.ConversationID, &userID, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if date.Valid { n.Date = &date.Time }
		if userID.Valid { id := int(userID.Int64); n.UserID = &id }
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (r *noteRepository) Create(ctx context.Context, q DBTX, note *Note) (int, error) {
	query := `INSERT INTO notes (tenant_id, text, date, conversation_id, user_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query, note.TenantID, note.Text, note.Date, note.ConversationID, note.UserID, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *noteRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error {
	query := `UPDATE notes SET text=$1, date=$2, updated_at=$3 WHERE id=$4 AND tenant_id=$5`
	result, err := q.ExecContext(ctx, query, note.Text, note.Date, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("note not found")
	}
	return nil
}

func (r *noteRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
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
