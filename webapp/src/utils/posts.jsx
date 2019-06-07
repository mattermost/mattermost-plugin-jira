// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// TODO: change to use the isCombinedUserActivityPost in 'mattermost-redux/utils/post_list' when we upgrade to 5.12
export const isCombinedUserActivityPost = (id) => {
    return (/^user-activity-(?:[^_]+_)*[^_]+$/).test(id);
};

