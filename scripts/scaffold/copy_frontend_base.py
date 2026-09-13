import shutil, os

src = r'e:\EnglishEncoding\competition\last\edms\sources\frontend'
dst = r'e:\EnglishEncoding\competition\last\MedTrust\frontend'

# 拷贝配置文件
for fname in ['package.json', 'vite.config.ts', 'tsconfig.json', 'index.html']:
    s = os.path.join(src, fname)
    d = os.path.join(dst, fname)
    if os.path.exists(s):
        shutil.copy2(s, d)
        print('Copied:', fname)

# 拷贝 assets (包含 theme.css)
src_assets = os.path.join(src, 'src', 'assets')
dst_assets = os.path.join(dst, 'src', 'assets')
if os.path.exists(src_assets):
    shutil.copytree(src_assets, dst_assets, dirs_exist_ok=True)
    print('Copied assets')

print('Frontend base files copied!')
