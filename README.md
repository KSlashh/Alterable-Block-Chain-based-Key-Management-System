# Alterable-Block-Chain-based-Key-Management-System
Revoke-friendly, alterable but not editable, only be able to make specific modifications .Based on chameleon hash , the consensus algorithm uses the raft algorithm.

Use Config to create config file(consists of parameter for chameleon hash). 

Use Client to send request to server. 

Use Server to create servers to handle requests and act as blockchain nodes. 

Need package [github.com/gorilla/mux](https://github.com/gorilla/mux) and [github.com/goraft/raft](https://github.com/goraft/raft)

Chameleon hash based on [julwil/chameleon_hash](https://github.com/julwil/chameleon_hash)
