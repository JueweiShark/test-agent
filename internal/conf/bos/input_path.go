package bos

import (
    "reflect"
    "strings"
)

const (
    UsecaseExportObjectName      = "{group_id}/{date_time}/export/{date_time}-{uuid}.xlsx"                                                  // 用例导出  vfleet-tmp/export/{group_id}/{user_id}/{date_time}/{uuid}.xlsx
    UsecaseGtExportObjectName    = "{group_id}/{date_time}/export/{uuid}.zip"                                                               // 用例GT导出 vfleet-tmp/export/{group_id}/{user_id}/{date_time}/{uuid}.zip
    TaskResultDiffObjectName     = "wsi/export/{date_time}/{uuid}/{date_time}-{task_ids}.xlsx"                                              // 导出对比结果 vfleet-tmp/export/{group_id}/{user_id}/{date_time}/{task_ids}-{uuid}.xlsx
    TaskResultKpiObjectName      = "{group_id}/{date_time}/export/{task_id}-{uuid}.zip"                                                     // 任务导出-数值型（kpi）vfleet-tmp/export/{group_id}/{user_id}/{date_time}/{task_id}-{uuid}.zip
    TaskResultBoolObjectName     = "{group_id}/{date_time}/export/{uuid}/{create_time}-{task_id}.xlsx"                                      // 任务导出-bool型 vfleet-tmp/export/{group_id}/{user_id}/{date_time}/{task_id}-{uuid}.xlsx
    KafkaTaskMetricLogObjectName = "{metric_logurl}/{uuid}.zip"                                                                             // 解析metric日志zip存储 vfleet-task/{group_id}/{user_id}/{date_time}/{task_id}/{sub_task_id}/XXXXXX.log
    KafkaUsecaseGtObjectName     = "vfleet-metrics/{usecase_dir_unbucket}/{origin_file_name}"                                               // GT入库 vfleet-metrics/groundturth/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.json
    KafkaUsecaseObjectName       = "vfleet-scenes/logsim/{group_id}/{1sid}/{2sid}/{3sid}/{origin_file_name_unext}-{uuid}.{origin_file_ext}" // DAT入库 vfleet-scenes/logsim/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.dat
    EgoBase64ToImage             = "wsi/resources/cover/{date_time}/{uuid}.{origin_file_ext}"                                               // 主车缩略图base64转存图片入库（已废弃） vfleet-assert/cars/{group_id}/{user_id}/XXXXXX.jpg
    ElementBase64ToImage         = "wsi/resources/cover/{date_time}/{uuid}.{origin_file_ext}"                                               // 元素缩略图base64转存图片入库（已废弃） vfleet-assert/elements/{group_id}/{user_id}/XXXXXX.jpg
    MetricTemplateGeneralKpi     = "{group_id}/generalKpi/{date_time}/{date_time}-{uuid}.config"                                            // 指标管理上传通用指标代码块 vfleet-metrics/general/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.config
    UsecaseProperKpi             = "{group_id}/properKpi/{date_time}/{date_time}-{uuid}.config"                                             // 用例编辑上传专用指标代码块 vfleet-metrics/special/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.config
    UsecaseDescribe              = "{group_id}/{date_time}/describe/{usecase_id}/{uuid}.json"                                               // 用例上传拼接好的描述信息 vfleet-scenes/logsim/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.json
    SceneSetBase64ToImage        = "wsi/resources/cover/{date_time}/{uuid}.{origin_file_ext}"                                               // 场景集缩略图base64转存图片入库（已废弃） vfleet-scenes/worldsim/{group_id}/XXXXXX.jpg
    MaterialsUploadFile          = "wsi/resources/{date_time}/{uuid}/{origin_file_name}"                                                    // 元素文件上传 vfleet-assert/elements/{group_id}/{user_id}/XXXXXX
    UsecaseVtdObjectName         = "vfleet-scenes/worldsim/{group_id}/{1sid}/{2sid}/{3sid}/{uuid}/{origin_file_name}"                       // worldsim的文件导入 vfleet-scenes/worldsim/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.xodr
    Task3DGSObjectName           = "vfleet-scenes/worldsim/{group_id}/{task_id}/{uuid}/{origin_file_name}"                                  // worldsim的文件导入 vfleet-scenes/worldsim/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.xodr
    TadSimXoscObjectName         = "vfleet-scenes/worldsim/tadSim/{usecase_id}/{uuid}/{origin_file_name}"                                   // worldsim的文件导入 vfleet-scenes/worldsim/{group_id}/1级目录id/2级目录id/3级目录id/XXXXXX.xodr

)

type ReplaceNums struct {
    GroupId             string `mapper:"{group_id}"`
    DateTime            string `mapper:"{date_time}"`
    Uuid                string `mapper:"{uuid}"`
    TaskIds             string `mapper:"{task_ids}"`
    TaskId              string `mapper:"{task_id}"`
    CreateTime          string `mapper:"{create_time}"`
    MetricLogurl        string `mapper:"{metric_logurl}"`
    UsecaseDirUnbucket  string `mapper:"{usecase_dir_unbucket}"`
    OriginFileName      string `mapper:"{origin_file_name}"`
    FirstLevelId        string `mapper:"{1sid}"`
    SecondLevelId       string `mapper:"{2sid}"`
    ThirdLevelId        string `mapper:"{3sid}"`
    OriginFileNameUnext string `mapper:"{origin_file_name_unext}"`
    OriginFileExt       string `mapper:"{origin_file_ext}"`
    UsecaseId           string `mapper:"{usecase_id}"`
}

func ReplaceAll(replacedString string, replacements *ReplaceNums) string {
    val := reflect.ValueOf(replacements).Elem()
    typ := val.Type()
    for i := 0; i < val.NumField(); i++ {
        replacedString = strings.Replace(
            replacedString,
            typ.Field(i).Tag.Get("mapper"),
            val.Field(i).String(),
            -1,
        )
    }
    return replacedString
}
