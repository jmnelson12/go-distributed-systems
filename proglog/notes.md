# Notes

At the beginning of this section, I had you put the log.proto file into an api/v1
directory. The v1 represents these protobufs’ major version. If you were to
continue building this project and decided to break API compatibility, you
would create a v2 directory to package the new messages together and communicate
to your users you’ve made incompatible API changes.
