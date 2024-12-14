### CHAT WEB-APPLICATION

This project is all about building a chat app that can handle lots of users and keep running smoothly, even if there are issues. The goals include making sure it can handle high traffic, grow easily by adding more servers, stay reliable, keep data in sync, and remember user sessions. To manage data like chat channels across different servers and allow for flexible scaling without too much disruption, I’m using something called consistent hashing.

For the back-end, I’m using SQLite3 to handle our databases, and we’re using WebSocket’s to make sure updates happen in real-time. Plus, we're relying on Kubernetes to manage all the containers. All these pieces work together to create a solid and fast chat platform that can support a lot of users without lagging.
