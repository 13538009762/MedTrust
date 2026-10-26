/**
 * 审计日志、区块链与系统看板监控模型定义
 */

export interface AuditLog {
  id: number
  user_id: number
  operation_type: string
  target_type: string
  target_id: string
  hospital_id: number
  result: 'SUCCESS' | 'FAILED' | 'INTERCEPTED'
  risk_level: 'LOW' | 'MEDIUM' | 'HIGH'
  ip_address: string
  user_name?: string
  hospital_name?: string
  created_at: string
}

export interface BlockchainTransaction {
  id: number
  tx_id: string
  business_type: string
  business_no: string
  block_number: number
  status: string
  created_at: string
}

export interface BlockchainStatus {
  is_live: boolean
  chain_type: string
  channel_name: string
  chaincode_name: string
  block_height: number
  tx_count: number
  active_nodes: number
  last_block_hash?: string
  last_updated?: string
}

export interface HospStat {
  name: string
  count: number
}

export interface RiskStat {
  risk_level: string
  count: number
}

export interface DayTrend {
  date: string
  flow_count: number
  intercept_count: number
}

export interface DashboardOverview {
  total_records: number
  total_emergencies: number
  pending_audits: number
  total_logs: number
  total_tx: number
  block_height: number
  hosp_stats: HospStat[]
  risk_stats: RiskStat[]
  throughput_trend: DayTrend[]
  total_intercepted: number
  tampered_count: number
  verified_count: number
  verification_pass_rate: number
}

export interface HospitalFlowItem {
  source: string
  target: string
  value: number
}

export interface AuthFunnelStats {
  total: number
  handled: number
  approved: number
  active: number
}

export interface NotificationItem {
  id: number
  type: string
  title: string
  message: string
  created_at: string
  payload?: any
}

export interface VerificationResult {
  record_id: number
  record_no: string
  is_tampered: boolean
  local_hash: string
  blockchain_hash: string
  integrity_status: string
  verify_timestamp: string
  details?: string
}
