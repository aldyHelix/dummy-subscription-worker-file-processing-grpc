package copyfto

import (
	_ "github.com/golang-migrate/migrate/v4/database/cockroachdb"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func PrepareDB() {
	// if dbConnPool != nil {
	// 	return
	// }

	// dbHost := os.Getenv("DB_HOST")
	// dbPort := os.Getenv("DB_PORT")
	// dbUser := os.Getenv("DB_USER")
	// dbPass := os.Getenv("DB_PASS")
	// dbName := os.Getenv("DB_NAME")
	// certPath := os.Getenv("DB_CERT_PATH")
	// migrationPath := "file://" + os.Getenv("MIGRATION_PATH")
	// if migrationPath == "file://" {
	// 	log.Println("MIGRATION_PATH environment is not set, using default value")
	// 	cwd, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	// 	migrationPath = "file://" + cwd + "/migrations/test"
	// }

	// url := fmt.Sprintf("cockroach://%s:%s@%s:%s/%s?sslmode=require&sslrootcert=test%s", dbUser, dbPass, dbHost, dbPort, dbName, certPath)
	// if os.Getenv("LOCAL_TEST") == "1" {
	// 	url = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	// }
	// log.Println(fmt.Sprintf("Migrating %s from %s", url, migrationPath))
	// m, err := migrate.New(
	// 	migrationPath,
	// 	url)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = m.Drop()
	// if err != nil && err.Error() != "no change" {
	// 	log.Fatal(err)
	// }

	// m, err = migrate.New(
	// 	migrationPath,
	// 	url)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = m.Up()
	// if err != nil && err.Error() != "no change" {
	// 	log.Fatal(err)
	// }

	// connString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=require&sslrootcert=%s", dbUser, dbPass, dbHost, dbPort, dbName, certPath)
	// log.Println(connString)
	// if os.Getenv("LOCAL_TEST") == "1" {
	// 	connString = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	// }
	// pgxConf, err := pgx.ParseConnectionString(connString)

	// if err != nil {
	// 	log.Panicln(err)
	// 	return
	// }

	// pgxPollConf := pgx.ConnPoolConfig{
	// 	ConnConfig:     pgxConf,
	// 	MaxConnections: 5,
	// }

	// dbConnPool, err = pgx.NewConnPool(pgxPollConf)
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }
}

type TestFixtures struct {
	Value string
	name  string
}

func (c *TestFixtures) SetValue(s string) {
	c.Value = s
}

func (c *TestFixtures) GetValue() string {
	return c.Value
}

func CreateFixtures(name string) *TestFixtures {
	f := TestFixtures{Value: "", name: name}

	return &f

}
