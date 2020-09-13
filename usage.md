# Run Systemd In Docker

Show you how to run systemd in docker using runc hooks and existing projects.

## Problem

The best article to understand the problem and challenge of running systemd is
[this][systemd in container].

## Solutions

Use [`oci-add-hooks`][oci-add-hooks] to wrap up `runc` and inject the
[`oci-systemd-hook-go`][oci-systemd-hook-go] hook that will do the necessary setup
for `systemd` to run in a container. See a more detailed description at the end
of `Step 3` below.

## Steps

1. Build and Install `oci-add-hooks` runtime
2. Build and Install `oci-systemd-hook-go` hook.
3. Config docker with `oci-add-hooks` runtime

### Step 1. Build and Install `oci-add-hooks`

Go to [oci-add-hooks][oci-add-hooks], build and install it.

Install the binary to `/usr/local/sbin/oci-add-hooks`.

You can install it other other locations, or leave it in `~/go/bin/`, but this
is the location we'll be using in the following configration.

### Step 2. Build and Install `oci-systemd-hook-go`.

Go to [oci-systemd-hook-go][oci-systemd-hook-go], build and install it.

Install the binary to `/usr/local/sbin/oci-systemd-hook-go`.

You can install it other other locations, or leave it in `~/go/bin/`, but this
is the location we'll be using in the following configration.

### Step 3. Config Docker with `oci-add-hook` runtime

Docker support multiple runtimes and when you do docker run you can specify
which runtime you want to run the container with, as you will see later.

First, config the Docker with the `oci-add-hook` runtime we build and installed
in step 1 above.

```
$cat /etc/docker/daemon.json
{
    "runtimes": {
        "oci-add-hooks": {
            "path": "/user/local/sbin/oci-add-hooks",
            "runtimeArgs": [
                "--hook-config-path",
                "/etc/docker/oci-hook-config.json",
                "--runtime-path",
                "/usr/local/sbin/runc"]
        }
    }
}
```

And, the content of `oci-hook-config.json`:

```
$cat /etc/docker/oci-hook-config.json
{
  "hooks": {
    "createContainer": [
      {
        "path": "/usr/local/sbin/oci-systemd-hook-go"
      }
    ]
  }
}
```

This is tell the `oci-add-hooks` runtime to *inject* above fragment into the
`config.json` file in the container bundle when Docker asks the runtime to run
the container. After the injection, `oci-add-hooks` will delegate the job of
actually running the container to `runc`, which is specified in the
`--runtime-path` for the `oci-add-hooks` args. When it is `runc`'s turn, before
running the container, it will check the `hooks` section and it see
`oci-systemd-hook-go` hook. That hook will be called to set up the stuff necessary
for the systemd to run.

## See It Works

- 1. Build a systemd container with httpd service running
- 2. Run with default runtime and see it fail
- 3. Run with the "new config" and see it success

### Build a test image

Build a docker image with following content, `docker build -t systemd-httpd .`

```
FROM fedora:latest
ENV container docker
RUN echo 'root:root' | chpasswd
RUN yum -y update && yum -y install httpd && yum clean all
RUN systemctl mask dnf-makecache.timer && systemctl enable httpd
CMD [ "/sbin/init" ]
```

Or you can just use an existing image:

```
docker pull binc/systemd-httpd:latest

```

### Run it with "vanilla" config

```
$ docker run -ti binc/systemd-httpd
Failed to mount tmpfs at /run: Operation not permitted
[!!!!!!] Failed to mount API filesystems.
Exiting PID 1...
```

### Run it with `oci-add-hooks` runtime and `oci-systemd-hook`
```
docker run --runtime=oci-add-hooks --stop-signal=RTMIN+3 -it --name systemd binc/systemd-httpd
```

You should be able to see it start successfully and you can login in with
root:root and check the service status:

```
[  OK  ] Reached target Multi-User System.
[  OK  ] Reached target Graphical Interface.
         Starting Update UTMP about System Runlevel Changes...
[  OK  ] Finished Update UTMP about System Runlevel Changes.

07753bf4c712 login: root
Password:root
[root@07753bf4c712 ~]# systemctl status httpd
● httpd.service - The Apache HTTP Server
     Loaded: loaded (/usr/lib/systemd/system/httpd.service; enabled; vendor preset: disabled)
     Active: active (running) since Wed 2020-09-02 03:29:29 UTC; 14s ago
       Docs: man:httpd.service(8)
   Main PID: 24 (httpd)
     Status: "Total requests: 0; Idle/Busy workers 100/0;Requests/sec: 0; Bytes served/sec:   0 B/sec"
     CGroup: /docker/07753bf4c71235f2a9c1088dad2cfa8bc1e7864038a864925da031d3be2b6b65/system.slice/httpd.service
             ├─24 /usr/sbin/httpd -DFOREGROUND
             ├─31 /usr/sbin/httpd -DFOREGROUND
             ├─32 /usr/sbin/httpd -DFOREGROUND
             ├─33 /usr/sbin/httpd -DFOREGROUND
             └─35 /usr/sbin/httpd -DFOREGROUND
```

You can stop it and `docker stop systemd` and see a clean stop/shutdown.

```
3c37f341295f login:
3c37f341295f login: [  OK  ] Removed slice system-getty.slice.
[  OK  ] Removed slice system-modprobe.slice.
[  OK  ] Stopped target Graphical Interface.
[  OK  ] Stopped target Multi-User System.
[  OK  ] Stopped target Login Prompts.
[  OK  ] Stopped target Timers.
[  OK  ] Stopped Daily Clean… Temporary Directories.
         Stopping Console Getty...
         Stopping The Apache HTTP Server...
         Stopping Home Area Manager...
         Stopping Login Service...
[  OK  ] Stopped Console Getty.
         Stopping Permit User Sessions...
[  OK  ] Stopped Permit User Sessions.
[  OK  ] Stopped Home Area Manager.
[  OK  ] Stopped Login Service.
[  OK  ] Stopped The Apache HTTP Server.
[  OK  ] Stopped target Basic System.
[  OK  ] Stopped target Paths.
[  OK  ] Stopped Dispatch Pa…onsole Directory Watch.
[  OK  ] Stopped Forward Pas…o Wall Directory Watch.
[  OK  ] Stopped target Remote File Systems.
[  OK  ] Stopped target Slices.
[  OK  ] Removed slice User and Session Slice.
[  OK  ] Stopped target Sockets.
         Stopping D-Bus System Message Bus...
[  OK  ] Stopped D-Bus System Message Bus.
         Stopping Update UTM…System Boot/Shutdown...
[  OK  ] Stopped Update UTMP…t System Boot/Shutdown.
[  OK  ] Stopped Create Vola… Files and Directories.
[  OK  ] Stopped target Local File Systems.
         Unmounting /etc/hostname...
         Unmounting /etc/hosts...
         Unmounting /etc/resolv.conf...
         Unmounting /run/lock...
         Unmounting Temporary Directory (/tmp)...
         Unmounting /var/log…6c16c35f3ce0f35892a1...
[  OK  ] Stopped Create System Users.
[FAILED] Failed unmounting /etc/hostname.
[FAILED] Failed unmounting /etc/hosts.
[FAILED] Failed unmounting /etc/resolv.conf.
[FAILED] Failed unmounting /run/lock.
[FAILED] Failed unmounting /…5f6c16c35f3ce0f35892a1.
         Unmounting /var/log/journal...
[FAILED] Failed unmounting T…orary Directory (/tmp).
[FAILED] Failed unmounting /var/log/journal.
[  OK  ] Stopped target Swap.
[  OK  ] Reached target Shutdown.
[  OK  ] Reached target Unmount All Filesystems.
[  OK  ] Reached target Final Step.
         Starting Halt...
$
```

[systemd in container]: https://developers.redhat.com/blog/2016/09/13/running-systemd-in-a-non-privileged-container/
[oci-add-hooks]: https://github.com/awslabs/oci-add-hooks
[oci-systemd-hook]: https://github.com/projectatomic/oci-systemd-hook
[oci-systemd-hook-go]: https://github.com/pierrchen/oci-systemd-hook-go
