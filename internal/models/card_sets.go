package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CardSet struct {
	ID          int
	Title       string
	Created     time.Time
	CardsNumber int
	Cards       Card
}

type CardSetModel struct {
	DB *pgxpool.Pool
}

func (m *CardSetModel) New(id int, title string) *CardSet {
	return &CardSet{
		ID:    id,
		Title: title,
	}
}

func (m *CardSetModel) Insert(ctx context.Context, title string, cardsNumber int) (int, error) {
	var id int

	query := `INSERT INTO card_sets (title, cards_number, created)
	VALUES($1, $2, NOW())
	RETURNING card_set_id`

	err := m.DB.QueryRow(ctx, query, title, cardsNumber).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *CardSetModel) Get(ctx context.Context, id int) (*CardSet, error) {
	query := `SELECT cs.card_set_id, title, created, cards_number, question, answer FROM card_sets cs
	join cards c using (card_set_id)
	WHERE cs.card_set_id = $1`

	var s = &CardSet{}
	rows, err := m.DB.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	err = rows.Scan(&s.ID, &s.Title, &s.Created, &s.CardsNumber, &s.Cards.Question, &s.Cards.Answer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}

	return s, nil
}

func (m *CardSetModel) ListAll(ctx context.Context) ([]*CardSet, error) {
	query := `SELECT card_set_id, title, created FROM card_sets
	ORDER BY card_set_id DESC`

	rows, err := m.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cardSets := []*CardSet{}
	for rows.Next() {
		s := &CardSet{}
		err = rows.Scan(&s.ID, &s.Title, &s.Created)
		if err != nil {
			return nil, err
		}
		cardSets = append(cardSets, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return cardSets, nil
}
