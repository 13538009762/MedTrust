import client from './client'
import type {
  ApiEnvelope,
  BlockchainStatus,
  Hospital,
  Department,
  User,
  CreateHospitalReq,
  UpdateHospitalReq,
  CreateDepartmentReq,
  UpdateDepartmentReq,
  CreateUserReq,
} from '@/types'

export function getBlockchainStatusApi(): Promise<ApiEnvelope<BlockchainStatus>> {
  return client.get('/system/blockchain/status')
}

export function listHospitalsApi(): Promise<ApiEnvelope<Hospital[]>> {
  return client.get('/system/hospitals')
}

export function createHospitalApi(data: CreateHospitalReq): Promise<ApiEnvelope<Hospital>> {
  return client.post('/system/hospitals', data)
}

export function updateHospitalApi(id: number, data: UpdateHospitalReq): Promise<ApiEnvelope<Hospital>> {
  return client.put(`/system/hospitals/${id}`, data)
}

export function updateHospitalStatusApi(id: number, status: number | string): Promise<ApiEnvelope<any>> {
  return client.put(`/system/hospitals/${id}/status`, { status })
}

export function listDepartmentsApi(hospitalId?: number): Promise<ApiEnvelope<Department[]>> {
  const url = hospitalId ? `/system/departments?hospital_id=${hospitalId}` : '/system/departments'
  return client.get(url)
}

export function createDepartmentApi(data: CreateDepartmentReq): Promise<ApiEnvelope<Department>> {
  return client.post('/system/departments', data)
}

export function updateDepartmentApi(id: number, data: UpdateDepartmentReq): Promise<ApiEnvelope<Department>> {
  return client.put(`/system/departments/${id}`, data)
}

export function deleteDepartmentApi(id: number): Promise<ApiEnvelope<any>> {
  return client.delete(`/system/departments/${id}`)
}

export function listUsersApi(): Promise<ApiEnvelope<User[]>> {
  return client.get('/system/users')
}

export function createUserApi(data: CreateUserReq | Record<string, any>): Promise<ApiEnvelope<any>> {
  return client.post('/system/users', data)
}

export function updateUserStatusApi(id: number, status: string): Promise<ApiEnvelope<any>> {
  return client.put(`/system/users/${id}/status`, { status })
}

export function resetUserPasswordApi(id: number, newPassword?: string): Promise<ApiEnvelope<any>> {
  return client.post(`/system/users/${id}/reset-password`, { new_password: newPassword })
}

export function listDoctorsApi(hospitalId?: number): Promise<ApiEnvelope<User[]>> {
  const url = hospitalId ? `/system/doctors?hospital_id=${hospitalId}` : '/system/doctors'
  return client.get(url)
}
