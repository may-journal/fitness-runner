<!-- cspell:ignore sdkmanrc -->

### 2026.10.17.2011

- Fix: nested types of one name keep their own members, and a record's compact constructor counts as its constructor.
- Fix: a Java main method may span lines, a private one proves nothing, and a hexadecimal literal ends before a plus.
- Fix: a code view field stated type first may hold spaces only inside its type arguments.
- Fix: reusable CI also sets up the JDK a root `.sdkmanrc` or `.tool-versions` pins on a line starting with java.
