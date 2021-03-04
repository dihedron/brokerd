/*
 * Brokerd API
 *
 * This API allows to interact with the `brokerd` daemon in its three different capacities:  1. as a **key/value store for properties**, holding key/value pairs; 2. as a **Raft cluster member**, through cluster managemenet APIs thata llow to check the state and   interact with the cluster (e.g. moving the master to another node, forcing a sync-up of the   cluster nodes, etc.) 3. as a **relational store for virtual machines and network ports**, as reported by OpenStack via its    notification exchanges inside RabbitMQ.  The **Properties API** allows to manage the lifecycle of key/value pairs; the kes part can encode a pseudo-hierarchical organisation of the information by adopting conventional characters as field separators, the way it is usually done in Java properties files, e.g.: ``` key-part-1.key-part-2.key-part-3....key-part-N=value ```  where each part of the key (`key-part-X`) can encode some part of a taxonomy.  The **Cluster API** provides a way to interact with the Raft cluster that guarantees that the Finite State Machines (FSM) holding the state of the several `brokerd` instances running on different  OpenStack controller nodes are all kept in sync and moving in lock-step. Through the API it is possible  to check the health and the status (*leader*, *follower*) of the nodes in the cluster, move the cluster leadership from the current master to a different node, force a sync-up of the cluster nodes, trigger the snaphotting of the current FSM state, etc.  The **Store API** provides a way to interact with the SQLite database holding information about  Virtual Machines and Network Ports.
 *
 * API version: 1.0
 * Contact: support@example.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type Error struct {
	Code string `json:"code"`

	Message string `json:"message"`
}
