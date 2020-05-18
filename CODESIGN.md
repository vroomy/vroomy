# Signing vroomy

1. Prepare code sign certificate
   
   2.1 Open "Keychain Access" (in Applications -> Utilities) and select "login" in the left pane.
   
   2.2 Select Create a Certificate ( in KeyChain Access -> Certificate Assistant )
       
   2.3 Input your name (whatever you like) and select "Code Signing" for Certificate Type.
       Not required but the name is used later in a command line so it could be better to use easily distinguishable name here (I use vroomySigner here).
       
  Now you have a certificate to code sign.


2. Install new vroomy version (auto upgrade is supported in v0.3.0 and up

   `vroomy upgrade`

   OR

   `cd ~/go/src/github.com/vroomy/vroomy && git pull && go install -trimpath`

   Note, the latter will not set the version, and `vroomy version` will print "undefined"


3. Re-signing vroomy

   `sudo ~/go/src/github.com/vroomy/vroomy/bin/codesign vroomySigner ~/go/bin/vroomy`    (Replace the vroomy path if it is different.)


4. Certificate Expiration

   A self-signed root certificate has a short expire date (1 year).
   Please check expire date of your certificate `vroomySigner` in Keychain Access App.
   If it expires, please re-create it.

   Using an apple signed certificate (if you are an apple developer) will also work!
