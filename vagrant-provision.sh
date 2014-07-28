#!/usr/bin/env bash

echo "pacman -Sy --noconfirm"
pacman -Sy --noconfirm
echo "pacman -S --noconfirm archlinux-keyring"
pacman -S --noconfirm archlinux-keyring
echo "pacman -Su --noconfirm"
pacman -Su --noconfirm
echo "pacman -S --needed --noconfirm base-devel git llvm-ocaml ocaml-findlib clang go colordiff mercurial"
pacman -S --needed --noconfirm base-devel git llvm-ocaml ocaml-findlib clang go colordiff mercurial
echo "git clone https://github.com/kerneis/cil; cd cil; ./configure; make; make install"
git clone https://github.com/kerneis/cil; cd cil; ./configure; make; make install
echo "cp -pr /vagrant/cilext /tmp/cilext"
cp -pr /vagrant/cilext /tmp/cilext
echo "cd /tmp/cilext; ocamlbuild -use-ocamlfind -package cil flattener.cma flattener.cmxs; ocamlfind install flattener META _build/flattener.cma _build/flattener.cmxs"
cd /tmp/cilext; ocamlbuild -use-ocamlfind -package cil flattener.cma flattener.cmxs; ocamlfind install flattener META _build/flattener.cma _build/flattener.cmxs
echo "cp -pr /vagrant/compiler /tmp/compiler"
cp -pr /vagrant/compiler /tmp/compiler
echo "cd /tmp/compiler; make; make install"
cd /tmp/compiler; make; make install
echo "cd"
cd
mkdir -p /home/vagrant/gowork/src/github.com/tjim
ln -s /vagrant /home/vagrant/gowork/src/github.com/tjim/smpcc
cat >>/home/vagrant/.bashrc <<EOF
export GOPATH=$HOME/gowork
EOF
