# someip

[![GitHub last commit (branch)](https://img.shields.io/github/last-commit/0x2E/someip/build?label=%E6%9C%80%E6%96%B0%E6%9E%84%E5%BB%BA)](https://github.com/0x2E/someip/tree/build)

中国大陆地区 IPv4 + IPv6，由多个数据源合并而来，每 3 天的 01:00 自动更新。

下载地址：

- [cidr.txt](https://raw.githubusercontent.com/0x2E/someip/build/cidr.txt)
- [Country.mmdb](https://raw.githubusercontent.com/0x2E/someip/build/Country.mmdb)

如果不准，请先测试一下最新版 Country.mmdb: <https://mmdb.rook1e.com/>。

## 数据源

IPv4:

- <https://github.com/17mon/china_ip_list>
- <https://github.com/metowolf/iplist>

IPv6:

- <https://github.com/gaoyifan/china-operator-ip>

欢迎推荐更好的数据源。

## 自定义

参考 `build.sh`，传入你喜欢的数据源：

```shell
Usage of someip:
  -i, --source strings       CIDR source files
  -o, --cidr-output string   CIDR ouput (default "cidr.txt")
  -m, --mmdb-output string   MMDB ouput (default "Country.mmdb")
  -h, --help                 Show usage
```
