export interface PresetAvatar {
  name: string
  tag: string
  tagType: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  bg: string
  url: string
}

export function getRoleTagType(role?: string): 'primary' | 'success' | 'warning' | 'danger' | 'info' {
  if (role === 'doctor') return 'primary'
  if (role === 'patient') return 'success'
  if (role === 'supervisor') return 'warning'
  if (role === 'admin') return 'danger'
  return 'info'
}

export function getRoleName(role?: string): string {
  if (role === 'doctor') return '执业医生'
  if (role === 'patient') return '居民患者'
  if (role === 'supervisor') return '卫健监管专员'
  if (role === 'admin') return '系统管理员'
  return '系统用户'
}

// 8 款精心设计的医疗与行业角色专属 SVG 预设头像
export const presetAvatars: PresetAvatar[] = [
  {
    name: '主任医师 (男)',
    tag: '临床专家',
    tagType: 'primary',
    bg: 'linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%233b82f6"/><circle cx="50" cy="40" r="18" fill="%23ffedd5"/><path d="M32 36 Q50 20 68 36 Q62 26 50 26 Q38 26 32 36 Z" fill="%231e293b"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 51 54 47" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M44 60 L50 72 L56 60 Z" fill="%230284c7"/><path d="M38 60 C38 72 40 76 44 78" stroke="%230284c7" stroke-width="3" fill="none" stroke-linecap="round"/><circle cx="44" cy="79" r="3.5" fill="%230369a1"/></svg>'
  },
  {
    name: '专家医师 (女)',
    tag: '主治专家',
    tagType: 'success',
    bg: 'linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%2310b981"/><path d="M28 42 C28 26 72 26 72 42 C72 58 66 60 66 60 L34 60 C34 60 28 58 28 42 Z" fill="%23334155"/><circle cx="50" cy="40" r="17" fill="%23fef3c7"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 50 54 47" stroke="%23e11d48" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M45 60 L50 70 L55 60 Z" fill="%23059669"/><path d="M48 68 H52 M50 66 V70" stroke="%23ffffff" stroke-width="2" stroke-linecap="round"/></svg>'
  },
  {
    name: '青年骨干医生',
    tag: '住院/主治',
    tagType: 'primary',
    bg: 'linear-gradient(135deg, #f5f3ff 0%, #ede9fe 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%238b5cf6"/><circle cx="50" cy="40" r="18" fill="%23fed7aa"/><path d="M30 38 Q50 20 70 38 Q64 24 50 24 Q36 24 30 38 Z" fill="%230f172a"/><circle cx="43" cy="39" r="2.5" fill="%230f172a"/><circle cx="57" cy="39" r="2.5" fill="%230f172a"/><path d="M46 48 Q50 51 54 48" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M42 60 L50 72 L58 60 Z" fill="%237c3aed"/></svg>'
  },
  {
    name: '科室主任医师',
    tag: '资深主任',
    tagType: 'warning',
    bg: 'linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%236366f1"/><circle cx="50" cy="40" r="18" fill="%23ffedd5"/><path d="M31 38 Q50 22 69 38 Q62 26 50 26 Q38 26 31 38 Z" fill="%23475569"/><circle cx="43" cy="39" r="2" fill="%231e293b"/><circle cx="57" cy="39" r="2" fill="%231e293b"/><circle cx="43" cy="39" r="5" stroke="%23334155" stroke-width="1.5" fill="none"/><circle cx="57" cy="39" r="5" stroke="%23334155" stroke-width="1.5" fill="none"/><line x1="48" y1="39" x2="52" y2="39" stroke="%23334155" stroke-width="1.5"/><path d="M46 48 Q50 51 54 48" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23ffffff"/><path d="M46 60 L50 68 L54 60 Z" fill="%234f46e5"/></svg>'
  },
  {
    name: '卫健监管专员',
    tag: '审计监管',
    tagType: 'warning',
    bg: 'linear-gradient(135deg, #fff7ed 0%, #ffedd5 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%23f59e0b"/><circle cx="50" cy="39" r="18" fill="%23fed7aa"/><path d="M32 35 Q50 21 68 35 Q60 25 50 25 Q40 25 32 35 Z" fill="%231e293b"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 50 54 47" stroke="%23c2410c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M20 84 C20 65 30 58 50 58 C70 58 80 65 80 84 Z" fill="%231e293b"/><path d="M46 58 L50 70 L54 58 Z" fill="%23ffffff"/><path d="M48 64 L50 78 L52 64 Z" fill="%23d97706"/><circle cx="68" cy="72" r="7" fill="%23fbbf24"/><path d="M68 68 L70 71 L73 70 L71 73 L73 76 L69 75 L67 77 L67 74 L64 73 L67 71 Z" fill="%23b45309"/></svg>'
  },
  {
    name: '系统技术管理员',
    tag: '系统运维',
    tagType: 'danger',
    bg: 'linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%23ef4444"/><circle cx="50" cy="39" r="18" fill="%23ffedd5"/><path d="M30 36 Q50 22 70 36 Q62 25 50 25 Q38 25 30 36 Z" fill="%230f172a"/><circle cx="43" cy="39" r="2.5" fill="%230f172a"/><circle cx="57" cy="39" r="2.5" fill="%230f172a"/><path d="M46 47 Q50 50 54 47" stroke="%23b91c1c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M20 84 C20 65 30 59 50 59 C70 59 80 65 80 84 Z" fill="%23334155"/><path d="M42 66 L47 71 L42 76" stroke="%2322c55e" stroke-width="2.5" fill="none" stroke-linecap="round" stroke-linejoin="round"/><line x1="50" y1="76" x2="58" y2="76" stroke="%2322c55e" stroke-width="2.5" stroke-linecap="round"/></svg>'
  },
  {
    name: '居民就诊患者 (男)',
    tag: '健康档案',
    tagType: 'info',
    bg: 'linear-gradient(135deg, #ecfeff 0%, #cffafe 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%2306b6d4"/><circle cx="50" cy="39" r="18" fill="%23ffedd5"/><path d="M32 35 Q50 20 68 35 Q60 24 50 24 Q40 24 32 35 Z" fill="%23334155"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 51 54 47" stroke="%23ea580c" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%230891b2"/><circle cx="50" cy="72" r="5" fill="%23ec4899"/><path d="M48 70 C48 68 52 68 52 70 C52 72 50 74 50 74 C50 74 48 72 48 70 Z" fill="%23ffffff"/></svg>'
  },
  {
    name: '居民就诊患者 (女)',
    tag: '健康档案',
    tagType: 'info',
    bg: 'linear-gradient(135deg, #fdf2f8 0%, #fce7f3 100%)',
    url: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="48" fill="%23ec4899"/><path d="M26 44 C26 26 74 26 74 44 C74 62 66 62 66 62 L34 62 C34 62 26 62 26 44 Z" fill="%23475569"/><circle cx="50" cy="40" r="17" fill="%23ffedd5"/><circle cx="43" cy="39" r="2.5" fill="%231e293b"/><circle cx="57" cy="39" r="2.5" fill="%231e293b"/><path d="M46 47 Q50 50 54 47" stroke="%23e11d48" stroke-width="2" fill="none" stroke-linecap="round"/><path d="M22 84 C22 66 32 60 50 60 C68 60 78 66 78 84 Z" fill="%23db2777"/><path d="M46 68 Q50 64 54 68 Q50 74 46 68 Z" fill="%23ffffff"/></svg>'
  }
]
