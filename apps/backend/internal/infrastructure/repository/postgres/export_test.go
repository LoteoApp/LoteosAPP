package postgres

func SetAfterStatementInstallments(repository *CollectionRepository, hook func()) {
	repository.afterStatementInstallments = hook
}
