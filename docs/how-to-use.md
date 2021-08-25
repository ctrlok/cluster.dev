# How to Use cdev

Since the cluster.dev (cdev) is quite powerfull framework it could be operated in serveral modes.

## Deploy an infrastructures from existing Templates

This mode also known as **user mode** gives you ability to launch ready-to-use infrastructure from prepared templates just by adding your cloud credentials and setting variables (like name, zones, number of instances, etc..).
You don't need to know backgroud tooling like Terraform or Helm, its just simple as donwload and launch command. Here is the steps:

* Install cdev binary
* Choose and Download Template
* Set Cloud Credentials
* Define variables for the template
* run cdev and get the cloud infrastructure

## Create own infrastructure Template

In this mode you would be able to create own infrastructure templates. With own templates you'll be able to launch or copy environments (like dev/stage/prod) using same template.
You'll be able to develop and propagate changes together with your team members just using git.
To operate cdev in the **developer mode**, there should be some prerequisites. You need to undestand Terraform and how to work with it's modules, it would be also good, but not mandatory, if you familiar with `go-template` syntax or `Helm`.

The easyest way to start is to downloadclone sample template project like [AWS EKS](https://github.com/shalb/cdev-aws-eks)
and launch infrastructure from one of examples.
Then you can edit some required variables play around and
you can play with changing values in the template itself.

