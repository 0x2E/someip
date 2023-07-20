# someip

[![GitHub last commit (branch)](https://img.shields.io/github/last-commit/0x2E/someip/build?label=%E6%9C%80%E6%96%B0%E6%9E%84%E5%BB%BA)](https://github.com/0x2E/someip/tree/build)

中国大陆地区 IP，由多个数据源合并而来，每 3 天的 01:00 自动更新。

下载地址：

- [cidr.txt](https://raw.githubusercontent.com/0x2E/someip/build/cidr.txt)
- [Country.mmdb](https://raw.githubusercontent.com/0x2E/someip/build/Country.mmdb)

## 数据源

- https://github.com/17mon/china_ip_list
- https://github.com/metowolf/iplist

目前借鉴了 [Hackl0us/GeoIP2-CN](https://github.com/Hackl0us/GeoIP2-CN) 使用的数据源。但做了合并优化，原始数据约 90k 条，合并优化后约 6k 条。

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
