docker run -d --name minio --restart always -p 9002:9002 -p 9001:9001 -v D:\TEMP\minio\stg:/data -e "MINIO_ROOT_USER=D9Q2D6JQGW1MVCC98LQL" -e "MINIO_ROOT_PASSWORD=LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr" minio/minio server /data --address ":9002" --console-address ":9001"