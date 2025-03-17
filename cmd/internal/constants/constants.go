package constants

const (
	Registered  = "REGISTERED" // заказ зарегистрирован, но вознаграждение не рассчитано
	Invalid     = "INVALID"    // заказ не принят к расчёту, и вознаграждение не будет начислено
	Processing  = "PROCESSING" // расчёт начисления в процессе
	Processed   = "PROCESSED"  // расчёт начисления окончен
	NotRelevant = "NORELEVANT" // заказ не зарегистрирован в системе расчёта
	New         = "NEW"        // новый заказ, по которому был получеен статус `429`(превышено количество запросов к сервису) от Accrual Service
)
