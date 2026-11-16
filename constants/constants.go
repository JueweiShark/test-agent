package constants

// 仿真模式
const (
    Pnc_log_sim_open         = "pnc_log_sim_open"
    Pnc_log_sim_close        = "pnc_log_sim_close"
    Perception_log_sim_open  = "perception_log_sim_open"
    Pnc_world_sim            = "pnc_world_sim"
    GradingOnly              = "grading_only"
    E2E_Stage2_log_sim_close = "e2e_stage2_log_sim_close"
)

// 任务状态

const (
    StartCode       = 1
    EndCode         = 2
    FailCode        = 3
    UnstartEnum     = "unstart"
    DoingEnum       = "doing"
    StopEnum        = "stop"
    EndEnum         = "end"
    FailEnum        = "fail"
    PassedEnum      = "passed"
    NotPassEnum     = "not_pass"
    DoneEnum        = "done"
    StartEnum       = "start"
    ParseMetricEnum = "parse_fail"
)
