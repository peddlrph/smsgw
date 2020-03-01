package main

import (
	"database/sql"
	//"errors"
	//	"net/http"
)

type message struct {
	Id       string `json:"id"`
	Body     string `json:"body"`
	Msg_Box  string `json:"msg_box"`
	Address  string `json:"address"`
	DateTime string `json:"time"`
}

type paymentstatus struct {
	Secret  string `json:"secret"`
	Payment *status
}

type status struct {
	Id         string `json:"id"`
	ButtonID   string `json:"buttonId"`
	ButtonData string `json:"buttonData"`
	Status     string `json:"status"`
	TxID       string `json:"txid"`
	NtxID      string `json:"ntxid"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Satoshis   string `json:"satoshis"`
}

type buttondata struct {
	MobileNum string `json:"mobile_num"`
	Product   string `json:"product"`
}

func (ps *paymentstatus) postPaymentStatus(db *sql.DB) error {
	s := ps.Payment
	_, err := db.Exec("INSERT INTO status(id,button_id,button_data,status,txid,ntxid,amount,currency,satoshis) VALUES(?,?,?,?,?,?,?,?,?)", s.Id, s.ButtonID, s.ButtonData, s.Status, s.TxID, s.NtxID, s.Amount, s.Currency, s.Satoshis)

	if err != nil {
		return err
	}

	return nil
}

func (s *status) postStatus(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO status(id,button_id,button_data,status,txid,ntxid,amount,currency,satoshis) VALUES(?,?,?,?,?,?,?,?,?)", s.Id, s.ButtonID, s.ButtonData, s.Status, s.TxID, s.NtxID, s.Amount, s.Currency, s.Satoshis)

	if err != nil {
		return err
	}

	return nil
}

func (m *message) getLastMessage(db *sql.DB) error {
	return db.QueryRow("SELECT id,body,msg_box,address,time FROM messages ORDER by ID desc LIMIT 1").Scan(&m.Id, &m.Body, &m.Msg_Box, &m.Address, &m.DateTime)
}

func (m *message) getMessage(db *sql.DB) error {
	return db.QueryRow("SELECT body,msg_box,address,time FROM messages WHERE id=?", m.Id).Scan(&m.Body, &m.Msg_Box, &m.Address, &m.DateTime)
}

func (m *message) createMessage(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO messages(id,body,msg_box,address,time) VALUES(?,?,?,?,?)", m.Id, m.Body, m.Msg_Box, m.Address, m.DateTime)

	if err != nil {
		return err
	}

	_, _ = db.Exec("INSERT IGNORE INTO phonebook(mobile_number) VALUES(?)", m.Address)

	return nil
}

func getMessages(db *sql.DB, start, count int) ([]message, error) {
	rows, err := db.Query(
		"SELECT id, body,msg_box,address,time FROM messages LIMIT ? OFFSET ?",
		count, start)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	messages := []message{}

	for rows.Next() {
		var m message
		if err := rows.Scan(&m.Id, &m.Body, &m.Msg_Box, &m.Address, &m.DateTime); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	return messages, nil
}
