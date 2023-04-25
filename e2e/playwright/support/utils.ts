import {UserProfile} from '@mattermost/types/users';
import Client4 from '@mattermost/client/client4';

export const DEFAULT_WAIT_MILLIS = 500;

export const cleanUpBotDMs = async (client: Client4, userId: UserProfile['id'], botUsername: string) => {
    const bot = await client.getUserByUsername(botUsername);

    const userIds = [userId, bot.id];
    const channel = await client.createDirectChannel(userIds);
    const posts = await client.getPosts(channel.id);

    const deletePostPromises = Object.keys(posts.posts).map(client.deletePost);
    await Promise.all(deletePostPromises);
}

export const getSlackAttachmentLocatorId = (postId: string) => {
    return `#post_${postId} .attachment__body`;
}

export const getPostMessageLocatorId = (postId: string) => {
    return `#post_${postId} .post-message`;
}
