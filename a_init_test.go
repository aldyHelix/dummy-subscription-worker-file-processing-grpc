package copyfto

func createCopyfto() *CopyftoAPI {
	PrepareDB()
	c := CopyftoAPI{
		Fixtures: CreateFixtures("dummy-api"),
	}
	c.initDb()
	return &c
}
