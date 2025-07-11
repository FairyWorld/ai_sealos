import { filterAccessToken } from '@/services/backend/middleware/access';
import { ErrorHandler } from '@/services/backend/middleware/error';
import { bindPhoneGuard } from '@/services/backend/middleware/oauth';
import { filterPhoneVerifyParams, verifyCodeGuard } from '@/services/backend/middleware/sms';
import { cnVersionMiddleware } from '@/services/backend/middleware/version';
import { bindPhoneSvc } from '@/services/backend/svc/bindProvider';
import { enablePhoneSms } from '@/services/enable';
import { NextApiRequest, NextApiResponse } from 'next';

export default ErrorHandler(async function handler(req: NextApiRequest, res: NextApiResponse) {
  await cnVersionMiddleware()(req, res, async () => {
    if (!enablePhoneSms()) {
      throw new Error('SMS is not enabled');
    }
    await filterAccessToken(
      req,
      res,
      async ({ userUid }) =>
        await filterPhoneVerifyParams(req, res, async ({ phoneNumbers, code }) => {
          await verifyCodeGuard(
            phoneNumbers,
            code,
            'phone_bind'
          )(res, () =>
            bindPhoneGuard(phoneNumbers, userUid)(res, () =>
              bindPhoneSvc(phoneNumbers, userUid)(res)
            )
          );
        })
    );
  });
});
