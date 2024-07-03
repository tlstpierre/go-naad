## Go Tools for NAAD and CAP-CP Alert System

This project is meant to develop a set of packages for receiving and processing messages from the Canadian Emergency Alert System.  By chaining together various packages into a message pipeline, a fairly customizable EAS receiver can be built.  

- naad-xml provides structs to decode the XML message format, and methods to handle embedded or linked resources, validate messages, and otherwise work with the CAP-CP format.

- naad-tcp provides a tcp client to receive alert messages over the Internet.

- TODO naad-filter provides mechanisms to filter messages based on location data or other content.

- TODO naad-cache provides a mechanism to store previously received messages and correlate them with duplicated or updated messages.  This also de-duplicates messages when multiple stream sources are used for redundancy, and can handle message updates.  The cache can be used as a reference for the current status of a given alert through the alert lifecycle.  Useful as a back-end for any graphical front-end system that needs to retain the state of all active alerts.

- TODO naad-multicast will play out embedded audio or TTS from alert messages to a multicast audio stream compatible with most IP audio devices such as IP phone (paging mode), or IP paging amplifiers.  Can also stream the text.

- TODO naad-web provides a web front-end for active alerts.