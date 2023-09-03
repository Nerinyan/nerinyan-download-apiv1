# 파생 파일 일괄 삭제
find ./* -maxdepth 1 -type f \( -name '*nv*' -o -name '*nh*' -o -name '*ns*' -o -name '*nb*' \) -exec rm -f {} +

find ./* -maxdepth 1 -type f -name '*nv*' -exec rm -f {} +
find ./* -maxdepth 1 -type f -name '*nh*' -exec rm -f {} +
find ./* -maxdepth 1 -type f -name '*ns*' -exec rm -f {} +
find ./* -maxdepth 1 -type f -name '*nb*' -exec rm -f {} +

