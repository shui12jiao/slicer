# AMF Metrics
gnb

类型: Gauge

描述: 表示 gNodeB 的数量，即连接到 AMF 的 gNB 数目。

示例值: 1

fivegs_amffunction_mm_confupdate

类型: Counter

描述: 表示 AMF 发出的 UE 配置更新命令的数量。

示例值: 2

fivegs_amffunction_rm_reginitreq

类型: Counter

描述: 表示 AMF 接收到的初始注册请求数。

示例值: 33

fivegs_amffunction_rm_regemergreq

类型: Counter

描述: 表示 AMF 接收到的紧急注册请求数。

示例值: 0

fivegs_amffunction_mm_paging5greq

类型: Counter

描述: 表示 AMF 发起的 5G 寻呼流程数。

示例值: 0

fivegs_amffunction_rm_regperiodreq

类型: Counter

描述: 表示 AMF 接收到的周期性注册更新请求数。

示例值: 0

fivegs_amffunction_mm_confupdatesucc

类型: Counter

描述: 表示 AMF 成功完成 UE 配置更新的次数。

示例值: 0

fivegs_amffunction_rm_reginitsucc

类型: Counter

描述: 表示 AMF 成功完成的初始注册次数。

示例值: 2

fivegs_amffunction_amf_authreject

类型: Counter

描述: 表示 AMF 发送的鉴权拒绝消息数量。

示例值: 0

fivegs_amffunction_rm_regmobreq

类型: Counter

描述: 表示 AMF 接收到的移动性注册更新请求数。

示例值: 0

amf_session

类型: Gauge

描述: 当前活跃的 AMF 会话数。

示例值: 2

fivegs_amffunction_rm_regmobsucc

类型: Counter

描述: 表示 AMF 成功完成的移动性注册更新次数。

示例值: 0

fivegs_amffunction_amf_authreq

类型: Counter

描述: 表示 AMF 发送的鉴权请求数。

示例值: 4

fivegs_amffunction_rm_regemergsucc

类型: Counter

描述: 表示 AMF 成功完成的紧急注册次数。

示例值: 0

fivegs_amffunction_mm_paging5gsucc

类型: Counter

描述: 表示 AMF 成功完成的 5G 寻呼流程次数。

示例值: 0

ran_ue

类型: Gauge

描述: 表示当前 RAN（无线接入网）中活跃的 UE 数量。

示例值: 2

fivegs_amffunction_rm_regperiodsucc

类型: Counter

描述: 表示 AMF 成功完成的周期性注册更新请求数。

示例值: 0

fivegs_amffunction_rm_regtime

类型: Histogram

描述: 表示注册过程所用时间的直方图分布（单位：秒或毫秒，根据实际配置）。

示例: 数据分布，可用于计算平均注册时延。

fivegs_amffunction_rm_registeredsubnbr

类型: Gauge

描述: 表示每个 AMF 中注册状态的用户数量。

示例值:

{plmnid="00101", snssai="2-000002"}: 1

{plmnid="00101", snssai="1-000001"}: 1

fivegs_amffunction_rm_reginitfail

类型: Counter

描述: 表示 AMF 初始注册失败的次数，按失败原因标记。

示例值:

{cause="9"}: 29

{cause="90"}: 1

{cause="11"}: 2

fivegs_amffunction_rm_regmobfail

类型: Counter

描述: 表示 AMF 移动性注册更新失败的次数。

示例值: 未提供

fivegs_amffunction_rm_regperiodfail

类型: Counter

描述: 表示周期性注册更新请求失败的次数。

示例值: 未提供

fivegs_amffunction_rm_regemergfail

类型: Counter

描述: 表示紧急注册请求失败的次数。

示例值: 未提供

fivegs_amffunction_amf_authfail

类型: Counter

描述: 表示 AMF 收到的鉴权失败消息的次数，按原因划分。

示例值: {cause="21"}: 2

process_max_fds

类型: Gauge

描述: 表示进程允许打开的最大文件描述符数量。

示例值: 1048576

process_virtual_memory_max_bytes

类型: Gauge

描述: 表示进程可用的最大虚拟内存（字节数），-1 表示无限制。

示例值: -1

process_cpu_seconds_total

类型: Gauge

描述: 表示进程占用的用户和系统 CPU 总时间（秒）。

示例值: 2822

process_virtual_memory_bytes

类型: Gauge

描述: 表示进程当前的虚拟内存大小（字节）。

示例值: 1847205888

process_resident_memory_bytes

类型: Gauge

描述: 表示进程常驻内存大小（字节）。

示例值: 25231360

process_start_time_seconds

类型: Gauge

描述: 表示进程启动时间，自 Unix Epoch（秒）。

示例值: 167051

process_open_fds

类型: Gauge

描述: 表示进程当前打开的文件描述符数量。

示例值: 28

# SMF Metrics
gn_rx_createpdpcontextreq

类型: Counter

描述: 表示接收到的 GTPv1C CreatePDPContextRequest 消息数。

示例值: 0

gn_rx_deletepdpcontextreq

类型: Counter

描述: 表示接收到的 GTPv1C DeletePDPContextRequest 消息数。

示例值: 0

gtp1_pdpctxs_active

类型: Gauge

描述: 表示当前活动的 GTPv1 PDP 上下文数（对应 GGSN）。

示例值: 0

fivegs_smffunction_sm_n4sessionreport

类型: Counter

描述: 表示 SMF 发起的 N4 会话报告请求数。

示例值: 0

ues_active

类型: Gauge

描述: 表示当前活跃的用户设备（UE）数量。

示例值: 1

gtp2_sessions_active

类型: Gauge

描述: 表示当前活动的 GTPv2 会话数（对应 PGW）。

示例值: 0

gtp_node_gn_rx_parse_failed

类型: Counter

描述: 表示因解析失败而丢弃的 GTPv1C 消息数。

示例值: 0

s5c_rx_createsession

类型: Counter

描述: 表示接收到的 GTPv2C CreateSessionRequest 消息数。

示例值: 0

s5c_rx_deletesession

类型: Counter

描述: 表示接收到的 GTPv2C DeleteSessionRequest 消息数。

示例值: 0

gtp_new_node_failed

类型: Counter

描述: 表示分配新的 GTP（对端）节点失败的次数。

示例值: 0

s5c_rx_parse_failed

类型: Counter

描述: 表示因解析失败而丢弃的 GTPv2C 消息数。

示例值: 0

fivegs_smffunction_sm_n4sessionreportsucc

类型: Counter

描述: 表示 SMF 成功发送的 N4 会话报告次数。

示例值: 0

gtp_node_gn_rx_createpdpcontextreq

类型: Counter

描述: 再次统计接收到的 GTPv1C CreatePDPContextRequest 消息数（可能与指标1类似）。

示例值: 无明确数据

fivegs_smffunction_sm_n4sessionestabreq

类型: Counter

描述: 表示 SMF 发起的 N4 会话建立请求数。

示例值: 0

bearers_active

类型: Gauge

描述: 表示当前活动承载（Bearer）数。

示例值: 1

gn_rx_parse_failed

类型: Counter

描述: 表示因解析失败而丢弃的 GTPv1C 消息数。

示例值: 0

gtp_peers_active

类型: Gauge

描述: 表示当前活动的 GTP 对等体数量。

示例值: 0

gtp_node_gn_rx_deletepdpcontextreq

类型: Counter

描述: 表示接收到的 GTPv1C DeletePDPContextRequest 消息数（再次统计）。

示例值: 无明确数据

gtp_node_s5c_rx_parse_failed

类型: Counter

描述: 表示因解析失败而丢弃的 GTPv2C 消息数。

示例值: 无明确数据

gtp_node_s5c_rx_createsession

类型: Counter

描述: 表示接收到的 GTPv2C CreateSessionRequest 消息数。

示例值: 无明确数据

gtp_node_s5c_rx_deletesession

类型: Counter

描述: 表示接收到的 GTPv2C DeleteSessionRequest 消息数。

示例值: 无明确数据

fivegs_smffunction_sm_sessionnbr

类型: Gauge

描述: 表示 SMF 管理下活跃会话的数量，按 PLMN 和 SNSSAI 标签划分。

示例值: {plmnid="00101", snssai="2-000002"}: 1

fivegs_smffunction_sm_pdusessioncreationreq

类型: Counter

描述: 表示 SMF 发起的 PDU 会话创建请求数。

示例值:

{plmnid="", snssai=""}: 1

{plmnid="00101", snssai="2-000002"}: 1

fivegs_smffunction_sm_pdusessioncreationsucc

类型: Counter

描述: 表示 SMF 成功创建 PDU 会话的次数。

示例值: {plmnid="00101", snssai="2-000002"}: 1

fivegs_smffunction_sm_qos_flow_nbr

类型: Gauge

描述: 表示 SMF 中 QoS 流的数量，按 PLMN、SNSSAI 和 FiveQI 分类。

示例值: {plmnid="00101", snssai="2-000002", fiveqi="9"}: 1

fivegs_smffunction_sm_seid_session

类型: Gauge

描述: 表示每个 SEID 下活跃会话的数量。

示例值: {plmnid="00101", snssai="2-000002", seid="2459"}: 1

fivegs_smffunction_sm_n4sessionestabfail

类型: Counter

描述: 表示 SMF 发起的 N4 会话建立请求失败的次数。

示例值: 无明确数据

fivegs_smffunction_sm_pdusessioncreationfail

类型: Counter

描述: 表示 SMF 创建 PDU 会话失败的次数。

示例值: 无明确数据

process_max_fds

类型: Gauge

描述: 表示进程允许的最大打开文件描述符数。

示例值: 1048576

process_virtual_memory_max_bytes

类型: Gauge

描述: 表示进程最大可用虚拟内存（字节），-1 表示无限制。

示例值: -1

process_cpu_seconds_total

类型: Gauge

描述: 表示进程累计的 CPU 时间（秒）。

示例值: 2331

process_virtual_memory_bytes

类型: Gauge

描述: 表示当前进程的虚拟内存大小（字节）。

示例值: 3129823232

process_resident_memory_bytes

类型: Gauge

描述: 表示当前进程的常驻内存大小（字节）。

示例值: 30277632

process_start_time_seconds

类型: Gauge

描述: 表示进程启动时间（Unix 时间戳，秒）。

示例值: 180927

process_open_fds

类型: Gauge

描述: 表示当前打开的文件描述符数。

示例值: 27

# UPF Metrics
fivegs_ep_n3_gtp_indatapktn3upf

类型: Counter

描述: 表示 UPF N3 接口上接收到的 GTP 数据包数（入站）。

示例值: 0

fivegs_ep_n3_gtp_outdatapktn3upf

类型: Counter

描述: 表示 UPF N3 接口上发送的 GTP 数据包数（出站）。

示例值: 0

fivegs_upffunction_sm_n4sessionestabreq

类型: Counter

描述: 表示 UPF 发起的 N4 会话建立请求数。

示例值: 1

fivegs_upffunction_sm_n4sessionreport

类型: Counter

描述: 表示 UPF 发起的 N4 会话报告请求数。

示例值: 0

fivegs_upffunction_sm_n4sessionreportsucc

类型: Counter

描述: 表示 UPF 成功发送 N4 会话报告的次数。

示例值: 0

fivegs_upffunction_upf_sessionnbr

类型: Gauge

描述: 表示当前 UPF 中活跃的会话数。

示例值: 1

fivegs_ep_n3_gtp_indatavolumeqosleveln3upf

类型: Counter

描述: 表示 UPF N3 接口上按 QoS 分级统计的入站数据包总数据量。

示例值: 未提供

fivegs_ep_n3_gtp_outdatavolumeqosleveln3upf

类型: Counter

描述: 表示 UPF N3 接口上按 QoS 分级统计的出站数据包总数据量。

示例值: 未提供

fivegs_upffunction_sm_n4sessionestabfail

类型: Counter

描述: 表示 UPF 发起的 N4 会话建立请求失败的次数。

示例值: 未提供

fivegs_upffunction_upf_qosflows

类型: Gauge

描述: 表示 UPF 中 QoS 流的数量，用于统计不同业务（例如 streaming）的数据流数量。

示例值: {dnn="streaming"}: 1

fivegs_ep_n3_gtp_indatavolumen3upf_seid

类型: Counter

描述: 表示 UPF 按 SEID 统计的入站 GTP 数据包数据量。

示例值: 未提供

fivegs_ep_n3_gtp_outdatavolumen3upf_seid

类型: Counter

描述: 表示 UPF 按 SEID 统计的出站 GTP 数据包数据量。

示例值: 未提供

process_max_fds

类型: Gauge

描述: 表示 UPF 进程允许的最大文件描述符数。

示例值: 1048576

process_virtual_memory_max_bytes

类型: Gauge

描述: 表示 UPF 进程最大可用虚拟内存（字节）。

示例值: -1

process_cpu_seconds_total

类型: Gauge

描述: 表示 UPF 进程累计使用的 CPU 时间（秒）。

示例值: 900

process_virtual_memory_bytes

类型: Gauge

描述: 表示 UPF 进程当前的虚拟内存大小（字节）。

示例值: 288837632

process_resident_memory_bytes

类型: Gauge

描述: 表示 UPF 进程当前的常驻内存大小（字节）。

示例值: 11669504

process_start_time_seconds

类型: Gauge

描述: 表示 UPF 进程启动时间（Unix 时间戳，秒）。

示例值: 181724

process_open_fds

类型: Gauge

描述: 表示 UPF 进程当前打开的文件描述符数。

示例值: 12