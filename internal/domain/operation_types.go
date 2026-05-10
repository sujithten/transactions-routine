package domain

const OperationTypeInstallment = 2

// DebitOperationTypes are op types whose amounts must be stored as negative.
var DebitOperationTypes = map[int]struct{}{1: {}, 2: {}, 3: {}}
