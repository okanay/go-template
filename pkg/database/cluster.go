package database

import "database/sql"

type Cluster struct {
	Master  *sql.DB // Yazma işlemleri için (INSERT, UPDATE)
	Replica *sql.DB // Okuma işlemleri için (SELECT)
}

func NewCluster(masterDSN, replicaDSN string) (*Cluster, error) {
	master, err := NewPostgres(masterDSN)
	if err != nil {
		return nil, err
	}

	replica, err := NewPostgres(replicaDSN)
	if err != nil {
		return nil, err
	}

	return &Cluster{Master: master, Replica: replica}, nil
}
