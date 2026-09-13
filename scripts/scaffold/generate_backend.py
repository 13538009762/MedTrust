# generate_backend.py
import os

base = r'e:\EnglishEncoding\competition\last\MedTrustackend'

def write_file(rel_path, content):
    full_path = os.path.join(base, rel_path)
    os.makedirs(os.path.dirname(full_path), exist_ok=True)
    with open(full_path, 'w', encoding='utf-8') as f:
        f.write(content.strip() + '\n')
    print('Wrote:', rel_path)

print('Generator initialized')
