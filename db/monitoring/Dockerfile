FROM alpine:3.24
RUN apk add --no-cache pgbadger perl-text-csv_xs \
    && sed -i 's@^use File::Spec qw/ tmpdir /;@use File::Spec;@' /usr/bin/pgbadger
ENTRYPOINT ["pgbadger"]
