import type { PatientOption } from '../types/clinical'

/**
 * Filter selectable patients in the encounter search dropdown by name,
 * account username, medical card number, ID card, or phone number.
 */
export function filterPatientOptions(patients: PatientOption[], query: string): PatientOption[] {
  const kw = query.trim().toLowerCase()
  if (!kw) return patients

  return patients.filter((p) => {
    const fields = [p.real_name, p.username, p.user_no, p.id_card, p.phone]
    return fields.some((val) => typeof val === 'string' && val.toLowerCase().includes(kw))
  })
}
