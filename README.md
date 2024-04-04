# Jaeger-PostgresQL

![GitHub License](https://img.shields.io/github/license/robbert229/jaeger-postgresql)
![GitHub Release](https://img.shields.io/github/v/release/robbert229/jaeger-postgresql)
[![Star on GitHub](https://img.shields.io/github/stars/robbert229/jaeger-postgresql.svg?style=flat)](https://github.com/robbert229/jaeger-postresql/stargazers)
![GitHub contributors from allcontributors.org](https://img.shields.io/github/all-contributors/robbert229/jaeger-postgresql)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=flat&logo=postgresql&logoColor=white)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=flat&logo=kubernetes&logoColor=white)

PostgreSQL is a great general purpose. Jaeger-PostgreSQL is intended to allow jaeger
to use postgresql as its backing storage solution for projects with low traffic. 

## Install

Installation is done through the use of a helm chart. 

<!-- x-release-please-start-version -->
```
helm install myrelease oci://ghcr.io/robbert229/jaeger-postgresql/charts/jaeger-postgresql \
    --version v1.6.0 \
    --set database.url='postgresql://postgres:password@database:5432/jaeger'
```
<!-- x-release-please-end -->

```
# database connection options
database:
    # url to the database
    url: "postgresql://postgres:password@database:5432/jaeger" 
    
    # the maximum number of database connections 
    maxConns: 10 

# configuration options for the cleaner
cleaner:
    # when true the cleaner will ensure that spans older than a set age will
    # be deleted.
    enabled: true

    # go duration formatted duration indicating the maximum age of a span 
    # before the cleaner removes it.
    maxSpanAge: "24h" 
```

## Usage

The Helm chart will deploy a service with the same name as the helm release. 
The configuration of Jaeger to use jaeger-postgresql depends on how you 
deployed jaeger. Adding the following argument to the jaeger's services, along
with the acompanying environment variables to your jaeger services will 
configure jaeger to use Jaeger-PostgresQL for storage 

`--grpc-storage.server=jaeger-postgresql:12345`

`SPAN_STORAGE_TYPE="grpc-plugin"`

The official jaeger documentation is the best place to look for detailed instructions on using a external storage plugin. https://www.jaegertracing.io/docs/1.55/deployment/#storage-plugin

## Contributors ✨

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="http://blog.johnrowley.co"><img src="https://avatars.githubusercontent.com/u/3454480?v=4?s=100" width="100px;" alt="John Rowley"/><br /><sub><b>John Rowley</b></sub></a><br /><a href="https://github.com/robbert229/jaeger-postgresql/commits?author=robbert229" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/jozef-slezak"><img src="https://avatars.githubusercontent.com/u/16844103?v=4?s=100" width="100px;" alt="Jozef Slezak"/><br /><sub><b>Jozef Slezak</b></sub></a><br /><a href="https://github.com/robbert229/jaeger-postgresql/commits?author=jozef-slezak" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

## Legacy

This project started out as a simple fork of [Jozef Slezak's plugin of the same name](jozef-slezak/jaeger-postgresql), but was eventually completely rewritten. 
