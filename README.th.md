# boxdb (db-backup)

[English](README.md) | **ไทย**

เครื่องมือ CLI สำหรับอัพโหลดไฟล์ backup ฐานข้อมูลขึ้น S3 เขียนด้วย Go รองรับ Ubuntu/Linux

## การติดตั้ง (Ubuntu)

### วิธีที่แนะนำ: แพ็คเกจ .deb (apt)

```sh
wget https://github.com/tockzazazx/db-backup/releases/latest/download/boxdb_amd64.deb
sudo apt install ./boxdb_amd64.deb
```

หากต้องการติดตั้งเวอร์ชันเจาะจง (หรือ prerelease) ชื่อ tag กับชื่อไฟล์ต้องเป็น
เวอร์ชันเดียวกัน — การผสมลิงก์ `latest` กับชื่อไฟล์ที่มีเลขเวอร์ชันจะได้ 404:

```sh
VERSION=0.8.0   # <- ไม่ต้องมี "v"
wget "https://github.com/tockzazazx/db-backup/releases/download/v${VERSION}/boxdb_${VERSION}_amd64.deb"
sudo apt install "./boxdb_${VERSION}_amd64.deb"
```

การอัพเดทใช้วิธีเดียวกัน: ดาวน์โหลด .deb ตัวใหม่แล้ว `apt install` ซ้ำ
apt จะจัดการ upgrade ให้ ตรวจสอบหรือถอนการติดตั้งได้ด้วย:

```sh
apt list --installed | grep boxdb
sudo apt remove boxdb
```

### วิธีสำรอง: install script (wget)

สำหรับเครื่องที่ใช้ apt ไม่ได้ สคริปต์จะติดตั้ง binary ตัวล่าสุดไว้ที่
`/usr/local/bin`:

```sh
wget -qO- https://github.com/tockzazazx/db-backup/releases/latest/download/install.sh | bash
```

> **คำเตือน:** เลือกวิธีใดวิธีหนึ่งแล้วใช้วิธีนั้นตลอด สคริปต์ติดตั้งลง
> `/usr/local/bin` ส่วน .deb ติดตั้งลง `/usr/bin` และ PATH จะเจอ
> `/usr/local/bin` ก่อน — binary เก่าจากสคริปต์จะบังตัวใหม่จาก .deb
> ทำให้ `boxdb --version` แสดงเวอร์ชันเก่าตลอด ถ้าจะย้ายจากสคริปต์มาใช้
> .deb ให้ลบตัวเก่าก่อน:
>
> ```sh
> sudo rm /usr/local/bin/boxdb && hash -r
> ```

### ตรวจสอบการติดตั้ง

```sh
boxdb --version
which -a boxdb   # ควรเห็น path เดียวเท่านั้น
```

## การใช้งาน

```sh
boxdb --version   # แสดงเวอร์ชัน
boxdb config      # แสดง config S3 ที่บันทึกไว้
boxdb test        # ทดสอบการเชื่อมต่อ S3
boxdb upload      # อัพโหลดไฟล์ใหม่จาก paths ที่ตั้งไว้
boxdb list        # ดูโฟลเดอร์วันที่บน S3
boxdb list <วันที่> # ดูไฟล์ในโฟลเดอร์วันนั้น
boxdb schedule    # ดู / ติดตั้ง / ถอน การอัพโหลดอัตโนมัติ
```

## การตั้งค่า S3

บันทึก credential ของ S3 (เก็บต่อ user ที่ `~/.config/boxdb/config.json`
permission 0600 — ไม่ต้องใช้ sudo):

```sh
boxdb config \
  --endpoint https://s3.example.com \
  --access AKIA... \
  --secret secret... \
  --bucket my-backups \
  --folder ubuntu-server-01 \
  --paths /var/backups,/opt/data
```

- `--endpoint` ใส่ได้ทั้ง `host:port` หรือ URL เต็ม — ถ้าขึ้นต้นด้วย `https://`
  จะเปิด TLS ให้อัตโนมัติ (หรือระบุ `--ssl` เองก็ได้)
- `--folder` คือโฟลเดอร์ (object prefix) ใน bucket สำหรับเก็บไฟล์ของ
  เครื่องนี้ ถ้ายังไม่มีจะถูกสร้างให้อัตโนมัติ
- `--paths` คือรายการโฟลเดอร์บนเครื่อง (คั่นด้วย comma) ที่จะถูกอัพโหลด
  เข้าโฟลเดอร์นั้น

รัน `boxdb config` โดยไม่ใส่ flag เพื่อดูค่าปัจจุบัน (secret ถูก mask)
และแก้ทีละ field ได้โดยส่งเฉพาะ flag ที่ต้องการเปลี่ยน

ทดสอบการเชื่อมต่อ (ใช้ [MinIO Go client](https://github.com/minio/minio-go)):

```sh
boxdb test
# connecting to s3.example.com (bucket "my-backups", ssl=true)...
# OK: connection successful, bucket is accessible
# OK: folder "ubuntu-server-01" created in bucket
# OK: local path /var/backups
```

`boxdb test` จะสร้างโฟลเดอร์ให้ถ้ายังไม่มี และเตือนถ้า path บนเครื่อง
ไม่มีอยู่จริง ส่วน error จะแสดงชัดเจนทั้งกรณียังไม่ได้ตั้ง config,
endpoint ต่อไม่ได้, credential ผิด, หรือไม่มี bucket

## การอัพโหลด

`boxdb upload` กวาดทุกโฟลเดอร์ใน `paths` แล้วอัพโหลดไฟล์เข้า bucket
ที่ `<folder>/<วันที่อัพโหลด>/` เช่น `ubuntu-server-01/2026-07-08/db1.pg`

- อัพโหลดเฉพาะไฟล์ที่ไม่เคยอัพมาก่อน — ไฟล์นับว่า "เคยอัพแล้ว" เมื่อมี
  object ชื่อเดียวกันอยู่ใต้ `folder` ในโฟลเดอร์วันที่ใดก็ตาม
- ไฟล์ที่ถูกลบจากเครื่องจะไม่ถูกลบออกจาก bucket
- โฟลเดอร์ย่อยและไฟล์ที่ขึ้นต้นด้วยจุดจะถูกข้าม

```sh
boxdb upload
# upload: /var/backups/3.pg -> ubuntu-server-01/2026-07-08/3.pg (300.0 KB)
# skip:   2.pg (already uploaded)
# done: 1 uploaded, 1 skipped
```

ดูของที่เก็บไว้แล้วด้วย `boxdb list`:

```sh
boxdb list                # ดูโฟลเดอร์วันที่ทั้งหมด
boxdb list 2026-07-08     # ดูไฟล์ในโฟลเดอร์วันนั้น
# ubuntu-server-01/2026-07-08:
#   3.pg    300.0 KB  2026-07-08 14:46:25
# total: 1 files, 300.0 KB
```

## การอัพโหลดอัตโนมัติตามเวลา

รัน `boxdb upload` อัตโนมัติผ่าน systemd timer — รายวัน รายสัปดาห์
หรือรายเดือน (1 schedule ต่อเครื่อง):

```sh
sudo boxdb schedule --daily 03:00                    # ทุกวัน เวลา 03:00
sudo boxdb schedule --weekly saturday --at 03:00     # ทุกวันเสาร์
sudo boxdb schedule --weekly sat,sun --at 03:00      # หลายวันต่อสัปดาห์
sudo boxdb schedule --monthly 1 --at 03:00           # วันที่ 1 ของทุกเดือน
sudo boxdb schedule --monthly last --at 03:00        # วันสุดท้ายของทุกเดือน
boxdb schedule                                       # ดูสถานะ รอบถัดไป ผลรอบล่าสุด
sudo boxdb schedule --remove                         # ถอนการติดตั้ง
```

`--monthly` รับ 1-28 หรือ `last` — วันที่ 29-31 ถูกปฏิเสธโดยตั้งใจ
เพราะ systemd จะข้ามเดือนที่ไม่มีวันนั้นไปเงียบๆ (เดือนกุมภาพันธ์จะไม่มี
backup!) ส่วน `last` หมายถึงวันสุดท้ายของเดือนเสมอ ไม่ว่าจะเป็นวันที่
28, 29, 30 หรือ 31

หมายเหตุ:

- ตัว upload จะรันเป็น user ที่เรียก `sudo` (อ่านจาก `$SUDO_USER`)
  จึงใช้ config ที่ `~/.config/boxdb/config.json` ของ user นั้น —
  ต้องรัน `boxdb config` และ `boxdb test` ด้วย user นั้นก่อน
  การติดตั้ง schedule จะไม่ยอมทำถ้า user นั้นยังไม่มี config
- ถ้าอยู่ใน root shell ที่เข้ามาด้วย `sudo su` / `sudo -i` ตัวแปร
  `$SUDO_USER` จะยังชี้ user เดิมอยู่ — ใช้ `--user` ระบุตรงๆ ได้ เช่น
  `boxdb schedule --daily 03:00 --user root`
- มีการตั้ง `Persistent=true` ไว้: ถ้าเครื่องปิดอยู่ตอนถึงเวลา งานจะรัน
  ชดเชยทันทีหลังเปิดเครื่อง ไม่ถูกข้าม
- ดู log การรัน: `journalctl -u boxdb-upload.service`
- มีได้ 1 schedule ต่อเครื่อง การตั้งใหม่จะทับของเดิม

## โครงสร้างโปรเจค

```
.
├── cmd/
│   └── boxdb/          # entrypoint หลัก
├── internal/
│   ├── config/         # จัดการไฟล์ config (~/.config/boxdb/config.json)
│   ├── s3/             # MinIO client wrapper (check, upload, list)
│   └── schedule/       # ติดตั้ง/ดูสถานะ/ถอน systemd timer
├── scripts/
│   ├── build-deb.sh    # แพ็ค .deb จาก binary ที่ build แล้ว
│   └── install.sh      # สคริปต์ติดตั้งสำหรับผู้ใช้
├── .github/workflows/
│   └── release.yml     # build binaries + .deb เมื่อ push tag
├── go.mod
├── Makefile
└── README.md
```

## การพัฒนา

ต้องใช้ Go 1.26 ขึ้นไป

```sh
make build        # build สำหรับเครื่องปัจจุบัน (bin/boxdb)
make build-linux  # cross-compile linux amd64 + arm64
make deb          # สร้างแพ็คเกจ .deb (ต้องมี dpkg-deb คือรันบน Linux)
make test
make lint
```

## การออก Release

Push tag แล้ว GitHub Actions จะ build binaries กับแพ็คเกจ .deb
และแนบเข้า GitHub release ให้อัตโนมัติ:

```sh
git tag v0.9.0
git push origin v0.9.0
```

เลขเวอร์ชันถูกฝังเข้า binary ตอน build ผ่าน `-ldflags`
(`make build VERSION=v0.9.0`) — tag ที่มีขีด (เช่น `v0.9.0-beta.1`)
จะถูกออกเป็น prerelease และไม่ขยับลิงก์ `releases/latest`
