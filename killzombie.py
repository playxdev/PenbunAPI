import os
import sys
from pathlib import Path

# บังคับให้ Console ของ Windows รองรับ UTF-8 เพื่อไม่ให้เกิด Error cp874
if sys.stdout and hasattr(sys.stdout, 'reconfigure'):
    try:
        sys.stdout.reconfigure(encoding='utf-8')
    except Exception:
        pass

def kill_zone_identifiers():
    base_path = Path(__file__).parent.resolve()
    
    print(f"[*] Starting scan in: {base_path}")
    print("-" * 50)
    
    deleted_count = 0
    error_count = 0
    
    for target_file in base_path.rglob("*Zone.Identifier"):
        if target_file.is_file():
            # แปลงอักขระพิเศษ \uf03a กลับเป็น ':' ก่อนสั่ง Print 
            safe_name = target_file.name.replace('\uf03a', ':')
            # ทำ Fallback เพิ่มเติม เผื่อมีอักขระประหลาดอื่นๆ
            safe_name = safe_name.encode('utf-8', errors='replace').decode('utf-8')
            
            try:
                target_file.unlink()
                print(f"[SUCCESS] Deleted: {safe_name}")
                deleted_count += 1
            except Exception as e:
                print(f"[FAILED] Could not delete {safe_name}")
                error_count += 1

    print("-" * 50)
    print("[*] Operation completed.")
    print(f"    - Total deleted: {deleted_count} files")
    if error_count > 0:
        print(f"    - Failed: {error_count} files")

if __name__ == "__main__":
    kill_zone_identifiers()