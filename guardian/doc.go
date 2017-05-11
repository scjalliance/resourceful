// Package guardian provides a cooperative resource checkin/checkout system that
// limits the number of users or consumers of a finite resource, such as
// digital licenses or file locks.
//
// Resources are the individual assets or entitities that are guarded.
//
// Consumers are the consumers of those finite resources.
//
// Policies describe the conditions and restrictions on resources, including
// the maximum allocation and duration of leases.
//
// Leases are time-limited exclusive locks on resources that are acquired by
// consumers.
//
// In a typical usage scenario the guardian client locates a guardian server,
// requests a lease while supplying the desired resource and identifying the
// consumer, is granted a lease, utilizes the resource, then releases the
// lease (or lets it expire).
package guardian
