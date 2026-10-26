/**
 * 医疗机构与科室领域模型定义
 */

export interface Hospital {
  id: number
  hospital_no: string
  name: string
  level: string
  address: string
  status: number // 1-正常, 0-停用
  created_at?: string
  updated_at?: string
}

export interface Department {
  id: number
  dept_no: string
  hospital_id: number
  name: string
  description?: string
  created_at?: string
  updated_at?: string
}

export interface CreateHospitalReq {
  hospital_no: string
  name: string
  level?: string
  address?: string
}

export interface UpdateHospitalReq {
  name: string
  level?: string
  address?: string
}

export interface UpdateHospitalStatusReq {
  status: number
}

export interface CreateDepartmentReq {
  dept_no: string
  hospital_id?: number
  name: string
  description?: string
}

export interface UpdateDepartmentReq {
  name: string
  description?: string
}
