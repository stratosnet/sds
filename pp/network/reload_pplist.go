package network

//ScheduleReloadPPlist
//	Long: 	pp not activated
//	Medium: mining not yet started
//	Short: 	by default (mining)

//func ScheduleReloadPPlist() {
//	var future time.Duration
//	if setting.State != types.PP_ACTIVE {
//		future = RELOAD_PP_LIST_INTERVAL_LONG
//	} else if !setting.IsStartMining {
//		future = RELOAD_PP_LIST_INTERVAL_MEDIUM
//	} else {
//		future = RELOAD_PP_LIST_INTERVAL_SHORT
//	}
//	utils.DebugLog("scheduled to get pp-list after: ", future.Seconds(), "second")
//	ppPeerClock.AddJobWithInterval(future, GetPPListFromSP)
//}
